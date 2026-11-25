package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cyril-fl/callback/core/api/services/auth"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/testserver"
)

// call sends body as JSON with token, and decodes the answer into out when given.
func call(t *testing.T, server *testserver.Server, token, method, path string, body any, out any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if out != nil {
		if err := json.NewDecoder(response.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
	return response
}

// The web app's whole path: log in with the cookie, add a batch, list with filters, change and delete by filter.
func TestApplications(t *testing.T) {
	server := testserver.Serve(t)

	// The login sets the cookie; the token is in the body only when asked.
	login, err := http.Post(server.URL+"/api/login", "application/json", strings.NewReader(`{"password":"`+testserver.Password+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	var session struct{ Token string }
	json.NewDecoder(login.Body).Decode(&session)
	login.Body.Close()
	if login.StatusCode != http.StatusOK || session.Token != "" || len(login.Cookies()) != 1 || login.Cookies()[0].Name != auth.Cookie || !login.Cookies()[0].HttpOnly {
		t.Fatalf("login: %d, token %q, cookies %v", login.StatusCode, session.Token, login.Cookies())
	}
	token := server.Session

	var created domain.BatchResult
	response := call(t, server, token, "POST", "/api/applications", []map[string]any{
		{"company": "Acme", "location": "Paris", "applied_at": "2026-01-10"},
		{"company": "Globex", "location": "Lyon", "status": "interview", "applied_at": "2026-02-10"},
		{"company": "", "location": "Lille"},
	}, &created)
	if response.StatusCode != http.StatusCreated || created.Count != 2 || len(created.Failed) != 1 || *created.Failed[0].Index != 2 {
		t.Fatalf("create: %d %+v", response.StatusCode, created)
	}

	var list []map[string]any
	response = call(t, server, token, "GET", "/api/applications?status=applied,interview&fields=company&sort=company&limit=1", nil, &list)
	if response.StatusCode != http.StatusOK || response.Header.Get("X-Total-Count") != "2" || len(list) != 1 || list[0]["company"] != "Acme" || list[0]["location"] != nil {
		t.Fatalf("list: %d %s %v", response.StatusCode, response.Header.Get("X-Total-Count"), list)
	}

	var updated domain.BatchResult
	response = call(t, server, token, "PATCH", "/api/applications", map[string]any{
		"ops": []map[string]any{{"q": "acme", "set": map[string]any{"status": "offer", "rating": 5}}},
	}, &updated)
	if response.StatusCode != http.StatusOK || updated.Count != 1 {
		t.Fatalf("update: %d %+v", response.StatusCode, updated)
	}

	var stats struct {
		Total    int
		ByStatus map[string]int `json:"by_status"`
	}
	call(t, server, token, "GET", "/api/stats", nil, &stats)
	if stats.Total != 2 || stats.ByStatus["offer"] != 1 {
		t.Fatalf("stats: %+v", stats)
	}

	var deleted domain.BatchResult
	response = call(t, server, token, "DELETE", "/api/applications", map[string]any{}, &deleted)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete with no filter: %d, want 400", response.StatusCode)
	}
	response = call(t, server, token, "DELETE", "/api/applications", map[string]any{"status": []string{"offer"}, "summary": true}, &deleted)
	if response.StatusCode != http.StatusOK || deleted.Count != 1 || deleted.IDs != nil {
		t.Fatalf("delete: %d %+v", response.StatusCode, deleted)
	}

	for _, bad := range []string{"/api/applications?limit=x", "/api/applications?sort=password", "/api/applications?status=hired"} {
		if response := call(t, server, token, "GET", bad, nil, nil); response.StatusCode != http.StatusBadRequest {
			t.Errorf("GET %s: %d, want 400", bad, response.StatusCode)
		}
	}
	if response := call(t, server, token, "POST", "/api/applications", map[string]any{"company": "not a list"}, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("an object for a batch: %d, want 400", response.StatusCode)
	}
}

// Who gets what: no token nothing, an API token no token routes, a read-only token no write.
func TestAuth(t *testing.T) {
	server := testserver.Serve(t)
	api := server.Token(t, false)
	readOnly := server.Token(t, true)

	tests := []struct {
		name, token, method, path string
		body                      any
		status                    int
	}{
		{"no token", "", "GET", "/api/applications", nil, http.StatusUnauthorized},
		{"wrong token", "callback_nope", "GET", "/api/me", nil, http.StatusUnauthorized},
		{"session, tokens", server.Session, "GET", "/api/tokens", nil, http.StatusOK},
		{"api token, list", api, "GET", "/api/applications", nil, http.StatusOK},
		{"api token, tokens", api, "GET", "/api/tokens", nil, http.StatusForbidden},
		{"api token, new token", api, "POST", "/api/tokens", map[string]any{"name": "x"}, http.StatusForbidden},
		{"api token, logout", api, "POST", "/api/logout", nil, http.StatusForbidden},
		{"api token, purge", api, "DELETE", "/api/data", nil, http.StatusForbidden},
		{"api token, trash", api, "GET", "/api/applications/deleted", nil, http.StatusOK},
		{"api token, forget", api, "DELETE", "/api/applications/deleted", map[string]any{"all": true}, http.StatusForbidden},
		{"session, forget", server.Session, "DELETE", "/api/applications/deleted", map[string]any{"all": true}, http.StatusOK},
		{"read-only, list", readOnly, "GET", "/api/applications", nil, http.StatusOK},
		{"read-only, create", readOnly, "POST", "/api/applications", []map[string]any{{"company": "A", "location": "B"}}, http.StatusForbidden},
		{"read-only, delete", readOnly, "DELETE", "/api/applications", map[string]any{"q": "a"}, http.StatusForbidden},
		{"session, purge", server.Session, "DELETE", "/api/data", nil, http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := call(t, server, test.token, test.method, test.path, test.body, nil)
			if response.StatusCode != test.status {
				t.Errorf("%d, want %d", response.StatusCode, test.status)
			}
			if test.token == readOnly && response.Header.Get(auth.ReadOnlyHeader) != "true" {
				t.Errorf("no %s header", auth.ReadOnlyHeader)
			}
		})
	}

	var me struct {
		Kind     string
		ReadOnly bool `json:"read_only"`
	}
	call(t, server, readOnly, "GET", "/api/me", nil, &me)
	if me.Kind != "api" || !me.ReadOnly {
		t.Errorf("me = %+v", me)
	}

	// A token made, listed, revoked: it stops working.
	var made struct {
		Token    string
		APIToken domain.Token `json:"api_token"`
	}
	if response := call(t, server, server.Session, "POST", "/api/tokens", map[string]any{"name": "script", "expires_in": "7d"}, &made); response.StatusCode != http.StatusCreated || !strings.HasPrefix(made.Token, "callback_") {
		t.Fatalf("create token: %d %+v", response.StatusCode, made)
	}
	if response := call(t, server, server.Session, "DELETE", "/api/tokens/"+made.APIToken.ID, nil, nil); response.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke: %d", response.StatusCode)
	}
	if response := call(t, server, made.Token, "GET", "/api/me", nil, nil); response.StatusCode != http.StatusUnauthorized {
		t.Errorf("revoked token: %d, want 401", response.StatusCode)
	}

	// Logged out, the session stops working.
	if response := call(t, server, server.Session, "POST", "/api/logout", nil, nil); response.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: %d", response.StatusCode)
	}
	if response := call(t, server, server.Session, "GET", "/api/me", nil, nil); response.StatusCode != http.StatusUnauthorized {
		t.Errorf("after logout: %d, want 401", response.StatusCode)
	}
}

// Wrong passwords in a row make the address wait.
func TestLoginDelay(t *testing.T) {
	server := testserver.Serve(t)
	var last *http.Response
	for range 5 {
		response, err := http.Post(server.URL+"/api/login", "application/json", strings.NewReader(`{"password":"wrong"}`))
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		last = response
	}
	if last.StatusCode != http.StatusTooManyRequests || last.Header.Get("Retry-After") == "" {
		t.Errorf("fifth wrong password: %d, Retry-After %q; want 429", last.StatusCode, last.Header.Get("Retry-After"))
	}
}
