package mcphttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/cyril-fl/jat/core/services/testserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// bearer sends a token with every request of the MCP client, as a connector of the Claude app does.
type bearer string

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+string(b))
	return http.DefaultTransport.RoundTrip(r)
}

// serve puts Handler in front of the API of a new base.
func serve(t *testing.T) (*testserver.Server, string) {
	t.Helper()
	api := testserver.Serve(t)
	target, err := url.Parse(api.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(Handler(httputil.NewSingleHostReverseProxy(target), "test"))
	t.Cleanup(server.Close)
	return api, server.URL
}

// connect is a real MCP client of endpoint, sending token.
func connect(t *testing.T, endpoint, token string) *mcp.ClientSession {
	t.Helper()
	transport := &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: &http.Client{Transport: bearer(token)}}
	agent, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { agent.Close() })
	return agent
}

// TestHandler serves the tools over HTTP for an API token only, 401 with where to sign in otherwise.
func TestHandler(t *testing.T) {
	api, endpoint := serve(t)
	for name, token := range map[string]string{"none": "", "session": api.Session, "unknown": "jat_nope"} {
		request, _ := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		response, err := (&http.Client{Transport: bearer(token)}).Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s token: %d, want 401", name, response.StatusCode)
		}
		if want := `Bearer resource_metadata="http://` + request.URL.Host + `/.well-known/oauth-protected-resource"`; response.Header.Get("WWW-Authenticate") != want {
			t.Errorf("%s token: WWW-Authenticate = %q, want %q", name, response.Header.Get("WWW-Authenticate"), want)
		}
	}

	agent := connect(t, endpoint, api.Token(t, false))
	result, err := agent.CallTool(context.Background(), &mcp.CallToolParams{Name: "add_applications", Arguments: map[string]any{
		"items": []map[string]any{{"company": "Acme", "location": "Paris"}, {"company": "Globex", "location": "Lyon"}},
	}})
	if err != nil || result.IsError {
		t.Fatalf("add_applications: %v %+v", err, result)
	}
	result, err = agent.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_applications", Arguments: map[string]any{"count_only": true}})
	if err != nil || result.IsError {
		t.Fatalf("list_applications: %v %+v", err, result)
	}
	var listed struct{ Total int }
	data, _ := json.Marshal(result.StructuredContent)
	if json.Unmarshal(data, &listed); listed.Total != 2 {
		t.Errorf("list_applications = %s, want a total of 2", data)
	}
}

// A read-only token gets the reading tools only.
func TestHandlerReadOnly(t *testing.T) {
	api, endpoint := serve(t)
	tools, err := connect(t, endpoint, api.Token(t, true)).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		if !tool.Annotations.ReadOnlyHint {
			t.Errorf("read-only token: %s is listed, it writes", tool.Name)
		}
		names[tool.Name] = true
	}
	if !names["list_applications"] || !names["read_applications"] || names["add_applications"] {
		t.Errorf("read-only tools = %v, want list_applications and read_applications, not add_applications", names)
	}
}
