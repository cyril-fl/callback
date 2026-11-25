// Package auth lets through the requests with a valid token only, and tells the routes which one.
package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cyril-fl/jat/config"
	"github.com/cyril-fl/jat/core/api/services/respond"
	"github.com/cyril-fl/jat/core/domain"
	"github.com/cyril-fl/jat/core/services/process/readtoken"
	"github.com/uptrace/bun"
)

// Cookie holds the session token of the web app: HttpOnly, page scripts never see it.
var Cookie = config.Current.Name + "_session"

// ReadOnlyHeader is set to "true" on the answers to a read-only token: the MCP over HTTP leaves out its writing tools.
const ReadOnlyHeader = "Jat-Read-Only"

// public routes answer without a token.
var public = map[string]bool{"POST /api/login": true}

type key struct{}

// Current is the token of the request.
func Current(ctx context.Context) domain.Token {
	token, _ := ctx.Value(key{}).(domain.Token)
	return token
}

// New runs next with the request's token in its context: the cookie, or "Authorization: Bearer <token>" for the
// CLI, scripts and agents. Without a valid one, 401.
func New(db bun.IDB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if public[r.Method+" "+r.URL.Path] || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		process := readtoken.Process(r.Context(), db, Token(r))
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		token := process.GetResult()
		if token.Kind == domain.TokenSession {
			// readtoken started the session's lifetime again: the cookie follows, when there is one.
			if _, err := r.Cookie(Cookie); err == nil {
				SetCookie(w, Token(r), token.ExpiresAt)
			}
		} else if manages(r) {
			respond.Fail(w, http.StatusForbidden, "a token cannot manage the tokens nor log out: log in")
			return
		}
		if token.ReadOnly {
			w.Header().Set(ReadOnlyHeader, "true")
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				respond.Fail(w, http.StatusForbidden, "this token is read-only: it cannot change anything")
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), key{}, token)))
	})
}

// manages tells the routes refused to an API or OAuth token. The mux has not matched the route yet, so this goes by
// path; the mux redirects an unclean path (//, ..) instead of serving it.
func manages(r *http.Request) bool {
	path := r.URL.Path
	return path == "/api/logout" || path == "/api/tokens" || strings.HasPrefix(path, "/api/tokens/")
}

// Token is the request's token, "" without one.
func Token(r *http.Request) string {
	if bearer, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return bearer
	}
	if cookie, err := r.Cookie(Cookie); err == nil {
		return cookie.Value
	}
	return ""
}

// SetCookie gives the browser the session token until expires; a zero time deletes it.
// Secure works on http://localhost too: browsers count it as a secure context.
func SetCookie(w http.ResponseWriter, token string, expires time.Time) {
	cookie := &http.Cookie{Name: Cookie, Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, Expires: expires}
	if expires.IsZero() {
		cookie.MaxAge = -1
	}
	http.SetCookie(w, cookie)
}
