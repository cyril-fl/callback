// Package cmdserve is jat-server serve: the API, the web app next to it, and the MCP over HTTP on its own address.
package cmdserve

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cyril-fl/jat/core/api"
	"github.com/cyril-fl/jat/core/api/services/guard"
	"github.com/cyril-fl/jat/core/services/process/purgetokens"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
)

const (
	purgeInterval   = time.Hour
	shutdownTimeout = 10 * time.Second
)

// slowRequest is the duration past which withLog logs a request; a var for the tests.
var slowRequest = time.Second

// MCP mounts the MCP over HTTP and its OAuth on mux, the API's handler given for the tools to call in process.
// Set by main once those packages exist; nil serves no MCP.
var MCP func(mux *http.ServeMux, db bun.IDB, api http.Handler, version string)

// Serve runs the API on db until SIGINT or SIGTERM; requests in progress finish first.
func Serve(cobra *cobra.Command, db *bun.DB) error {
	flags := cobra.Flags()
	addr, _ := flags.GetString("addr")
	origin, _ := flags.GetString("origin")
	web, _ := flags.GetString("web")
	published, _ := flags.GetString("published")
	mcpAddr, _ := flags.GetString("mcp-addr")
	mcpPublished, _ := flags.GetString("mcp-published")
	if err := guard.CheckOrigin(origin); err != nil {
		return err
	}
	if published == "" {
		published = addr
	}
	if mcpPublished == "" {
		mcpPublished = mcpAddr
	}

	ctx, stop := signal.NotifyContext(cobra.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cleanup := func() {
		purge := purgetokens.Process(ctx, db)
		purge.Run()
		if purge.HasErr() && ctx.Err() == nil {
			log.Printf("purge tokens: %v", purge.GetErr())
		}
	}
	go func() {
		cleanup()
		every(ctx, purgeInterval, cleanup)
	}()

	handler := api.New(db, origin)
	if web != "" {
		handler = withWeb(web, handler)
	}
	if onLoopback(published) {
		handler = withProxy(handler)
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           withLog(handler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       time.Minute,
	}

	failed := make(chan error, 2)
	go func() { failed <- server.ListenAndServe() }()
	log.Printf("listening on %s", addr)
	servers := []*http.Server{server}
	if mcpAddr != "" && MCP != nil {
		// The API's handler, without the web app nor the proxy: the tools call it in process.
		mux := http.NewServeMux()
		MCP(mux, db, api.New(db, origin), cobra.Root().Version)
		var mcpHandler http.Handler = mux
		if onLoopback(mcpPublished) {
			mcpHandler = withProxy(mcpHandler)
		}
		mcpServer := &http.Server{
			Addr:              mcpAddr,
			Handler:           withLog(mcpHandler),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       time.Minute,
			WriteTimeout:      5 * time.Minute,
			IdleTimeout:       time.Minute,
		}
		go func() { failed <- mcpServer.ListenAndServe() }()
		log.Printf("serving MCP on %s/mcp, with its OAuth", mcpAddr)
		servers = append(servers, mcpServer)
	}

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	// Not ctx: it is already canceled, and requests in progress need time to finish.
	shutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	for _, server := range servers {
		if err := server.Shutdown(shutdown); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	log.Print("stopped")
	return nil
}

// every runs task at each interval until ctx ends.
func every(ctx context.Context, interval time.Duration, task func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task()
		}
	}
}

// onLoopback tells if only this machine reaches addr: then the X-Forwarded-For of a request can only come from a proxy
// running here (tailscale serve), never from a client.
func onLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// withProxy gives the requests the client's address: behind a proxy, RemoteAddr is the proxy's for every device, and
// the login delay would block them all. The last X-Forwarded-For entry is the one the proxy wrote, the client cannot
// forge it.
func withProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded := r.Header.Values("X-Forwarded-For")
		if len(forwarded) > 0 {
			entries := strings.Split(forwarded[len(forwarded)-1], ",")
			if address := strings.TrimSpace(entries[len(entries)-1]); address != "" {
				r.RemoteAddr = net.JoinHostPort(address, "0")
			}
		}
		next.ServeHTTP(w, r)
	})
}

// withLog logs the method, path, status and duration of the requests that fail (>= 500) or take slowRequest or more:
// one line per request would drown the logs, the app makes many.
func withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if took := time.Since(start); recorder.status >= 500 || took >= slowRequest {
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.status, took.Round(time.Millisecond))
		}
	})
}

// statusWriter keeps the status written through it.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Unwrap lets http.ResponseController reach the real writer: the import and the export lift their deadlines with it.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// withWeb serves the web app of dir next to the API, so both share one origin. A path that is no file is a page of
// the app: 200.html, the fallback `nuxt generate` writes, lets its router show it. A directory is never listed: only
// its index.html counts as a file, so /_nuxt/ gets the app, not the build's file list.
func withWeb(dir string, api http.Handler) http.Handler {
	files := http.FileServer(http.Dir(dir))
	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path
		if strings.HasSuffix(name, "/") {
			name += "index.html"
		}
		if f, err := http.Dir(dir).Open(name); err == nil {
			info, err := f.Stat()
			f.Close()
			if err == nil && !info.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, filepath.Join(dir, "200.html"))
	})
	return mux
}
