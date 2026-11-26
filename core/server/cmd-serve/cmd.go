// Package cmdserve is callback-server serve: the API, the web app next to it, and the MCP over HTTP on its own address.
package cmdserve

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/api"
	"github.com/cyril-fl/callback/core/api/services/guard"
	"github.com/cyril-fl/callback/core/domain"
	cmdadmin "github.com/cyril-fl/callback/core/server/cmd-admin"
	"github.com/cyril-fl/callback/core/services/process/markunanswered"
	"github.com/cyril-fl/callback/core/services/process/purgetokens"
	"github.com/cyril-fl/callback/core/services/process/readpreferences"
	"github.com/cyril-fl/callback/core/services/secret"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/uptrace/bun"
)

const (
	// purgeInterval is how often serve purges the expired tokens and marks the unanswered applications.
	purgeInterval   = time.Hour
	shutdownTimeout = 10 * time.Second
)

// slowRequest is the duration past which withLog logs a request; a var for the tests.
var slowRequest = time.Second

// MCP mounts the MCP over HTTP and its OAuth on mux, the API's handler given for the tools to call in process.
// Set by main once those packages exist; nil serves no MCP.
var MCP func(mux *http.ServeMux, db bun.IDB, api http.Handler, version string, mode domain.ServerMode)

// Serve runs the API on db until SIGINT or SIGTERM; requests in progress finish first. It first creates the account
// of an empty base (cmdadmin.EnsurePassword), and refuses the single mode on a base of several accounts.
func Serve(cobra *cobra.Command, db *bun.DB) error {
	flags := cobra.Flags()
	mode, err := serverMode(flags)
	if err != nil {
		return err
	}
	if err := secret.Check(); err != nil && !errors.Is(err, secret.ErrNoKey) {
		return err
	}
	if err := cmdadmin.EnsurePassword(cobra.Context(), db, mode.Multi()); err != nil {
		return err
	}
	if err := checkAccounts(cobra.Context(), db, mode); err != nil {
		return err
	}
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
		markUnanswered(ctx, db)
	}
	go func() {
		cleanup()
		every(ctx, purgeInterval, cleanup)
	}()

	handler := api.New(db, origin, mode)
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
	log.Printf("listening on %s, %s mode", addr, mode.Mode)
	servers := []*http.Server{server}
	if mcpAddr != "" && MCP != nil {
		// The API's handler, without the web app nor the proxy: the tools call it in process.
		mux := http.NewServeMux()
		MCP(mux, db, api.New(db, origin, mode), cobra.Root().Version, mode)
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

// serverMode reads --mode and --add-user-mode, or CALLBACK_MODE and CALLBACK_ADD_USER_MODE when the flag is not
// given.
func serverMode(flags *pflag.FlagSet) (domain.ServerMode, error) {
	mode := domain.ServerMode{Mode: flagOrEnv(flags, "mode", "MODE"), AddUserMode: flagOrEnv(flags, "add-user-mode", "ADD_USER_MODE")}
	switch {
	case !slices.Contains(domain.Modes, mode.Mode):
		return mode, fmt.Errorf("--mode must be one of %v, not %q", domain.Modes, mode.Mode)
	case !slices.Contains(domain.AddUserModes, mode.AddUserMode):
		return mode, fmt.Errorf("--add-user-mode must be one of %v, not %q", domain.AddUserModes, mode.AddUserMode)
	}
	return mode, nil
}

func flagOrEnv(flags *pflag.FlagSet, name, key string) string {
	value, _ := flags.GetString(name)
	if env := os.Getenv(config.Env(key)); env != "" && !flags.Changed(name) {
		return env
	}
	return value
}

// checkAccounts refuses the single mode on a base of several accounts: its login, the password alone, could not
// tell them apart.
func checkAccounts(ctx context.Context, db bun.IDB, mode domain.ServerMode) error {
	if mode.Multi() {
		return nil
	}
	count, err := db.NewSelect().Model((*domain.User)(nil)).Count(ctx)
	if err != nil {
		return err
	}
	if count > 1 {
		return fmt.Errorf("the base has %d accounts: serve it with --mode multi (%s=multi), the single mode holds one", count, config.Env("MODE"))
	}
	return nil
}

// markUnanswered moves to no_reply the applications each account's preferences say nobody will answer.
func markUnanswered(ctx context.Context, db bun.IDB) {
	var users []string
	if err := db.NewSelect().Model((*domain.User)(nil)).Column("id").Scan(ctx, &users); err != nil {
		if ctx.Err() == nil {
			log.Printf("list users: %v", err)
		}
		return
	}
	for _, user := range users {
		markUserUnanswered(domain.WithUser(ctx, user), db)
	}
}

func markUserUnanswered(ctx context.Context, db bun.IDB) {
	preferences := readpreferences.Process(ctx, db)
	preferences.Run()
	if preferences.HasErr() {
		if ctx.Err() == nil {
			log.Printf("read preferences: %v", preferences.GetErr())
		}
		return
	}
	mark := markunanswered.Process(ctx, db, preferences.GetResult().NoReplyDays, time.Now())
	mark.Run()
	switch {
	case mark.HasErr() && ctx.Err() == nil:
		log.Printf("mark unanswered: %v", mark.GetErr())
	case mark.GetResult() > 0:
		log.Printf("%d applications moved to no_reply", mark.GetResult())
	}
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
