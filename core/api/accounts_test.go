package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/testserver"
)

// An account never reaches another's data: B neither reads, changes, deletes, restores, exports, counts nor lists
// what A owns, by id or by filter, and B's purge leaves A's data alone.
func TestAccountsApart(t *testing.T) {
	server := testserver.Serve(t)
	a := server.Session
	var created domain.BatchResult
	call(t, server, a, "POST", "/api/applications", []map[string]any{{"company": "Acme", "location": "Paris"}, {"company": "Globex", "location": "Lyon"}}, &created)
	acme, globex := created.IDs[0], created.IDs[1]
	call(t, server, a, "DELETE", "/api/applications", map[string]any{"ids": []string{globex}}, nil)
	var made struct {
		APIToken domain.Token `json:"api_token"`
	}
	call(t, server, a, "POST", "/api/tokens", map[string]any{"name": "script"}, &made)
	token := made.APIToken
	if token.ID == "" {
		t.Fatal("no token made")
	}

	b := server.Account(t, "bob")
	for _, path := range []string{"/api/applications", "/api/applications?ids=" + acme, "/api/applications/deleted", "/api/export", "/api/stats", "/api/sources", "/api/status-changes", "/api/tokens"} {
		var body json.RawMessage
		response := call(t, server, b, "GET", path, nil, &body)
		if response.StatusCode != http.StatusOK || strings.Contains(string(body), acme) || strings.Contains(string(body), globex) || strings.Contains(string(body), token.ID) || strings.Contains(string(body), "Acme") {
			t.Errorf("B GET %s: %d %s", path, response.StatusCode, body)
		}
	}
	var stats struct{ Total int }
	if call(t, server, b, "GET", "/api/stats", nil, &stats); stats.Total != 0 {
		t.Errorf("B stats total = %d, want 0", stats.Total)
	}
	writes := []struct {
		method, path string
		body         any
	}{
		{"PATCH", "/api/applications", map[string]any{"ops": []map[string]any{{"ids": []string{acme}, "set": map[string]any{"status": "offer"}}}}},
		{"PATCH", "/api/applications", map[string]any{"ops": []map[string]any{{"q": "acme", "set": map[string]any{"status": "offer"}}}}},
		{"DELETE", "/api/applications", map[string]any{"ids": []string{acme}}},
		{"DELETE", "/api/applications", map[string]any{"q": "acme"}},
		{"POST", "/api/applications/restore", map[string]any{"ids": []string{globex}}},
		{"DELETE", "/api/applications/deleted", map[string]any{"all": true}},
	}
	for _, write := range writes {
		var result domain.BatchResult
		call(t, server, b, write.method, write.path, write.body, &result)
		if result.Count != 0 {
			t.Errorf("B %s %s %v: %+v", write.method, write.path, write.body, result)
		}
	}
	if response := call(t, server, b, "DELETE", "/api/tokens/"+token.ID, nil, nil); response.StatusCode != http.StatusNotFound {
		t.Errorf("B deletes A's token: %d, want 404", response.StatusCode)
	}
	if response := call(t, server, b, "DELETE", "/api/data", nil, nil); response.StatusCode != http.StatusOK {
		t.Errorf("B purge: %d", response.StatusCode)
	}

	var list []domain.Application
	call(t, server, a, "GET", "/api/applications", nil, &list)
	var trash []domain.DeletedApplication
	call(t, server, a, "GET", "/api/applications/deleted", nil, &trash)
	var tokens []domain.Token
	call(t, server, a, "GET", "/api/tokens", nil, &tokens)
	if len(list) != 1 || list[0].ID != acme || list[0].Status != domain.StatusApplied || len(trash) != 1 || trash[0].ID != globex || len(tokens) != 1 {
		t.Errorf("A after B: list %+v, trash %+v, tokens %+v", list, trash, tokens)
	}
}

// The account changes its own password given the current one: this session stays, the others log out, a token cannot.
func TestEditPassword(t *testing.T) {
	server := testserver.Serve(t)
	var login struct{ Token string }
	call(t, server, "", "POST", "/api/login", map[string]any{"password": testserver.Password, "token": true}, &login)

	if response := call(t, server, server.Session, "PATCH", "/api/me/password", map[string]any{"current": "wrong one", "new": "battery staple"}, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("wrong current: %d, want 400", response.StatusCode)
	}
	if response := call(t, server, server.Token(t, false), "PATCH", "/api/me/password", map[string]any{"current": testserver.Password, "new": "battery staple"}, nil); response.StatusCode != http.StatusForbidden {
		t.Errorf("API token: %d, want 403", response.StatusCode)
	}
	if response := call(t, server, server.Session, "PATCH", "/api/me/password", map[string]any{"current": testserver.Password, "new": "short"}, nil); response.StatusCode != http.StatusBadRequest {
		t.Errorf("short: %d, want 400", response.StatusCode)
	}
	if response := call(t, server, server.Session, "PATCH", "/api/me/password", map[string]any{"current": testserver.Password, "new": "battery staple"}, nil); response.StatusCode != http.StatusNoContent {
		t.Fatalf("change: %d, want 204", response.StatusCode)
	}
	if response := call(t, server, server.Session, "GET", "/api/me", nil, nil); response.StatusCode != http.StatusOK {
		t.Errorf("own session: %d, want 200", response.StatusCode)
	}
	if response := call(t, server, login.Token, "GET", "/api/me", nil, nil); response.StatusCode != http.StatusUnauthorized {
		t.Errorf("other session: %d, want 401", response.StatusCode)
	}
	if response := call(t, server, "", "POST", "/api/login", map[string]any{"password": "battery staple"}, nil); response.StatusCode != http.StatusOK {
		t.Errorf("login with the new one: %d", response.StatusCode)
	}
}
