// Package mcphttp serves the MCP tools of jat mcp over streamable HTTP, for an agent that cannot run jat where it
// runs (the Claude app calls from Anthropic's cloud).
package mcphttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	cmdmcp "github.com/cyril-fl/jat/cli/cmd-mcp"
	"github.com/cyril-fl/jat/cli/services/apiclient"
	"github.com/cyril-fl/jat/core/api/services/auth"
	"github.com/cyril-fl/jat/core/server/oauth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler serves the tools to the bearer of an API or OAuth token, "Authorization: Bearer <token>", the tools calling
// api in process with it. Without a valid one, 401 with where to sign in (oauth.Challenge): a session token is
// refused too, only a token can be revoked alone. A read-only token gets the reading tools only.
func Handler(api http.Handler, version string) http.Handler {
	options := &mcp.StreamableHTTPOptions{
		// No session kept between requests: each one carries its token, and a restart loses nothing.
		Stateless:    true,
		JSONResponse: true,
		// The token authenticates every request, so a DNS rebinding page gains nothing; and a tunnel or proxy on this
		// machine (Tailscale Funnel, cloudflared) forwards from the loopback with the public Host, which this would refuse.
		DisableLocalhostProtection: true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		// localhost: the API's guard takes this Host for this machine.
		client, err := apiclient.NewClientWithResponses("http://localhost", apiclient.WithHTTPClient(inProcess{api}), apiclient.Bearer(token))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ok, readOnly := valid(r.Context(), client)
		if token == "" || !ok {
			w.Header().Set("WWW-Authenticate", oauth.Challenge(r))
			http.Error(w, "an API token is required: Authorization: Bearer jat_…, or sign in with OAuth", http.StatusUnauthorized)
			return
		}
		server := cmdmcp.NewServer(client, readOnly, version)
		mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, options).ServeHTTP(w, r)
	})
}

// valid asks the API whether the token is known, unexpired and not a session's, before any tool: the client learns
// it at connection. The answer tells a read-only token too.
func valid(ctx context.Context, client *apiclient.ClientWithResponses) (ok, readOnly bool) {
	response, err := client.GetApiMe(ctx)
	me, err := apiclient.Decode[apiclient.LoginMeResult](response, err)
	if err != nil || me.Kind == apiclient.TokenSession {
		return false, false
	}
	return true, me.ReadOnly || response.Header.Get(auth.ReadOnlyHeader) == "true"
}

// inProcess runs the API's handler for the client's requests, without a round trip through the network.
type inProcess struct {
	api http.Handler
}

func (p inProcess) Do(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	p.api.ServeHTTP(w, r)
	return w.Result(), nil
}
