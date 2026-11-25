package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cyril-fl/callback/core/api/services/auth"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/testserver"
)

// A client from its discovery to a refresh: metadata, registration, the consent page and the token endpoint, as
// ChatGPT or Claude go through them, and its access token on the API.
func TestFlow(t *testing.T) {
	api := testserver.Serve(t)
	mux := http.NewServeMux()
	Routes(mux, api.DB)
	server := httptest.NewServer(mux)
	defer server.Close()
	// No redirect followed: the test reads where the page sends the browser.
	browser := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	getJSON := func(path string, dest any) {
		t.Helper()
		response, err := browser.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(dest) != nil {
			t.Fatalf("GET %s: %d", path, response.StatusCode)
		}
	}
	var resource struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}
	getJSON("/.well-known/oauth-protected-resource/mcp", &resource)
	if resource.Resource != server.URL+"/mcp" || len(resource.AuthorizationServers) != 1 || resource.AuthorizationServers[0] != server.URL {
		t.Errorf("protected resource = %+v, want %s/mcp signed in by %s", resource, server.URL, server.URL)
	}
	var metadata map[string]any
	getJSON("/.well-known/oauth-authorization-server", &metadata)
	if metadata["issuer"] != server.URL || metadata["token_endpoint"] != server.URL+"/token" || metadata["authorization_response_iss_parameter_supported"] != true {
		t.Errorf("authorization server = %v", metadata)
	}
	if response, err := browser.Get(server.URL + "/.well-known/openid-configuration"); err != nil {
		t.Fatal(err)
	} else if response.Body.Close(); response.StatusCode != http.StatusNotFound {
		t.Errorf("GET /.well-known/openid-configuration: %d, want 404", response.StatusCode)
	}
	preflight, _ := http.NewRequest(http.MethodOptions, server.URL+"/token", nil)
	if response, err := browser.Do(preflight); err != nil {
		t.Fatal(err)
	} else if response.Body.Close(); response.StatusCode != http.StatusNoContent || response.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("OPTIONS /token: %d, want 204 with CORS", response.StatusCode)
	}

	registerClient := func(body string) (int, string) {
		t.Helper()
		response, err := browser.Post(server.URL+"/register", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var registered struct {
			ClientID string `json:"client_id"`
		}
		json.NewDecoder(response.Body).Decode(&registered)
		return response.StatusCode, registered.ClientID
	}
	for _, uri := range []string{"http://evil.example/cb", "https://x.example/cb#frag", "javascript:alert(1)", "callback://cb", "https://user@x.example/cb"} {
		if status, _ := registerClient(`{"client_name":"x","redirect_uris":["` + uri + `"]}`); status != http.StatusBadRequest {
			t.Errorf("register %s: %d, want 400", uri, status)
		}
	}
	status, clientID := registerClient(`{"client_name":"ChatGPT","redirect_uris":["https://chatgpt.com/connector_platform_oauth_redirect","http://localhost/callback"],"token_endpoint_auth_method":"client_secret_post"}`)
	if status != http.StatusCreated || clientID == "" {
		t.Fatalf("register: %d %q, want 201 and a client_id", status, clientID)
	}

	verifier := strings.Repeat("a", 50)
	sum := sha256.Sum256([]byte(verifier))
	asked := url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {"https://chatgpt.com/connector_platform_oauth_redirect"},
		"state": {"xyz"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"},
	}
	authorize := func(method string, values url.Values) (*http.Response, string) {
		t.Helper()
		var response *http.Response
		var err error
		if method == http.MethodGet {
			response, err = browser.Get(server.URL + "/authorize?" + values.Encode())
		} else {
			response, err = browser.PostForm(server.URL+"/authorize", values)
		}
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return response, string(body)
	}
	with := func(changes ...string) url.Values {
		values := url.Values{}
		for key, value := range asked {
			values[key] = value
		}
		for i := 0; i < len(changes); i += 2 {
			values.Set(changes[i], changes[i+1])
		}
		return values
	}

	// A redirect the client did not register shows an error, and never goes there.
	for _, redirect := range []string{"https://evil.example/cb", "https://chatgpt.com/connector_platform_oauth_redirect/x", "http://localhost/other"} {
		if response, _ := authorize(http.MethodGet, with("redirect_uri", redirect)); response.StatusCode != http.StatusBadRequest || response.Header.Get("Location") != "" {
			t.Errorf("unregistered redirect %s: %d to %q, want 400 and no redirect", redirect, response.StatusCode, response.Header.Get("Location"))
		}
	}
	if response, _ := authorize(http.MethodGet, with("client_id", "nope")); response.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown client: %d, want 400", response.StatusCode)
	}
	// A request without PKCE S256 or with another response_type shows an error, never a redirect: registration is
	// open, that would be an open redirect.
	for name, value := range map[string][2]string{"no S256": {"code_challenge_method", "plain"}, "token flow": {"response_type", "token"}} {
		if response, _ := authorize(http.MethodGet, with(value[0], value[1])); response.StatusCode != http.StatusBadRequest || response.Header.Get("Location") != "" {
			t.Errorf("%s: %d to %q, want 400 and no redirect", name, response.StatusCode, response.Header.Get("Location"))
		}
	}
	response, body := authorize(http.MethodGet, asked)
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "ChatGPT") || !strings.Contains(body, "chatgpt.com") || response.Header.Get("X-Frame-Options") != "DENY" {
		t.Errorf("consent page: %d, want 200 naming ChatGPT and chatgpt.com, unframable", response.StatusCode)
	}
	// The loopback matches whatever the port, as Claude Code picks one at each sign-in.
	if response, body := authorize(http.MethodGet, with("redirect_uri", "http://localhost:5555/callback")); response.StatusCode != http.StatusOK || !strings.Contains(body, "localhost:5555") {
		t.Errorf("loopback redirect on another port: %d, want the page", response.StatusCode)
	}

	location := func(response *http.Response) url.Values {
		t.Helper()
		target, err := url.Parse(response.Header.Get("Location"))
		if err != nil || response.StatusCode != http.StatusFound || !strings.HasPrefix(target.String(), "https://chatgpt.com/connector_platform_oauth_redirect?") {
			t.Fatalf("redirect: %d to %q", response.StatusCode, response.Header.Get("Location"))
		}
		return target.Query()
	}
	denied := location(func() *http.Response { r, _ := authorize(http.MethodPost, with("decision", "deny")); return r }())
	if denied.Get("error") != "access_denied" || denied.Get("state") != "xyz" || denied.Get("iss") != server.URL {
		t.Errorf("deny: %v, want access_denied with the state and the issuer", denied)
	}
	if response, body := authorize(http.MethodPost, with("decision", "allow", "password", "wrong")); response.StatusCode != http.StatusUnauthorized || !strings.Contains(body, "Wrong password") {
		t.Errorf("wrong password: %d, want 401 and the page again", response.StatusCode)
	}

	tokenRequest := func(values url.Values) (int, map[string]any) {
		t.Helper()
		response, err := browser.PostForm(server.URL+"/token", values)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var out map[string]any
		json.NewDecoder(response.Body).Decode(&out)
		return response.StatusCode, out
	}
	// connect allows the client and exchanges the code: the tokens of a new connection.
	connect := func(readOnly bool) map[string]any {
		t.Helper()
		changes := []string{"decision", "allow", "password", testserver.Password}
		if readOnly {
			changes = append(changes, "read_only", "on")
		}
		allowed := location(func() *http.Response { r, _ := authorize(http.MethodPost, with(changes...)); return r }())
		if allowed.Get("code") == "" || allowed.Get("state") != "xyz" || allowed.Get("iss") != server.URL {
			t.Fatalf("allow: %v, want a code with the state and the issuer", allowed)
		}
		exchange := url.Values{"grant_type": {"authorization_code"}, "code": {allowed.Get("code")}, "client_id": {clientID},
			"redirect_uri": {asked.Get("redirect_uri")}, "code_verifier": {verifier}, "client_secret": {"ignored"}}
		status, tokens := tokenRequest(exchange)
		if status != http.StatusOK || tokens["token_type"] != "Bearer" || tokens["refresh_token"] == nil {
			t.Fatalf("exchange: %d %v", status, tokens)
		}
		if status, out := tokenRequest(exchange); status != http.StatusBadRequest || out["error"] != "invalid_grant" {
			t.Errorf("code used twice: %d %v, want invalid_grant", status, out)
		}
		return tokens
	}
	// me is GET /api/me with token: the status, the answer and whether it says read-only.
	me := func(token string) (int, map[string]any, bool) {
		t.Helper()
		request, _ := http.NewRequest(http.MethodGet, api.URL+"/api/me", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var out map[string]any
		json.NewDecoder(response.Body).Decode(&out)
		return response.StatusCode, out, response.Header.Get(auth.ReadOnlyHeader) == "true"
	}

	tokens := connect(false)
	if status, out, readOnly := me(tokens["access_token"].(string)); status != http.StatusOK || out["kind"] != string(domain.TokenOAuth) || out["name"] != "ChatGPT" || readOnly {
		t.Errorf("GET /api/me: %d %v read-only %v, want 200, an oauth token named ChatGPT, not read-only", status, out, readOnly)
	}
	if status, _, readOnly := me(connect(true)["access_token"].(string)); status != http.StatusOK || !readOnly {
		t.Errorf("read-only connection: %d read-only %v, want 200 and %s: true", status, readOnly, auth.ReadOnlyHeader)
	}

	// A wrong verifier spends the code and gets nothing.
	allowed := location(func() *http.Response {
		r, _ := authorize(http.MethodPost, with("decision", "allow", "password", testserver.Password))
		return r
	}())
	if status, out := tokenRequest(url.Values{"grant_type": {"authorization_code"}, "code": {allowed.Get("code")}, "client_id": {clientID},
		"redirect_uri": {asked.Get("redirect_uri")}, "code_verifier": {strings.Repeat("b", 50)}}); status != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Errorf("wrong verifier: %d %v, want invalid_grant", status, out)
	}

	refresh := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens["refresh_token"].(string)}, "client_id": {clientID}}
	status, fresh := tokenRequest(refresh)
	if status != http.StatusOK || fresh["access_token"] == tokens["access_token"] {
		t.Fatalf("refresh: %d %v, want new tokens", status, fresh)
	}
	if status, out := tokenRequest(refresh); status != http.StatusBadRequest || out["error"] != "invalid_grant" {
		t.Errorf("old refresh token: %d %v, want invalid_grant", status, out)
	}
	if status, _, _ := me(tokens["access_token"].(string)); status != http.StatusUnauthorized {
		t.Errorf("old access token after the refresh: %d, want 401", status)
	}
	if status, _, _ := me(fresh["access_token"].(string)); status != http.StatusOK {
		t.Errorf("new access token: %d, want 200", status)
	}
	if status, out := tokenRequest(url.Values{"grant_type": {"password"}, "client_id": {clientID}}); status != http.StatusBadRequest || out["error"] != "unsupported_grant_type" {
		t.Errorf("password grant: %d %v, want unsupported_grant_type", status, out)
	}
	if status, out := tokenRequest(url.Values{"grant_type": {"refresh_token"}, "client_id": {"nope"}}); status != http.StatusUnauthorized || out["error"] != "invalid_client" {
		t.Errorf("unknown client: %d %v, want invalid_client", status, out)
	}

	// The wrong passwords of the page wait like the login's: the fifth in a row waits, and so does the login.
	var last *http.Response
	for range 5 {
		last, _ = authorize(http.MethodPost, with("decision", "allow", "password", "wrong"))
	}
	if last.StatusCode != http.StatusTooManyRequests || last.Header.Get("Retry-After") == "" {
		t.Errorf("wrong passwords in a row: %d, want 429 with Retry-After", last.StatusCode)
	}
	login, err := http.Post(api.URL+"/api/login", "application/json", strings.NewReader(`{"password":"`+testserver.Password+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	login.Body.Close()
	if login.StatusCode != http.StatusTooManyRequests {
		t.Errorf("login after the consent's wrong passwords: %d, want 429: they share the count", login.StatusCode)
	}
}
