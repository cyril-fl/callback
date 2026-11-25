// Package api is the HTTP API: one folder per route, each running a process of core/services/process.
package api

import (
	"net/http"

	apicreateapplications "github.com/cyril-fl/callback/core/api/create-applications"
	apideleteapplications "github.com/cyril-fl/callback/core/api/delete-applications"
	apieditapplications "github.com/cyril-fl/callback/core/api/edit-applications"
	apilogin "github.com/cyril-fl/callback/core/api/login"
	apireadapplications "github.com/cyril-fl/callback/core/api/read-applications"
	apireadstats "github.com/cyril-fl/callback/core/api/read-stats"
	"github.com/cyril-fl/callback/core/api/services/auth"
	"github.com/cyril-fl/callback/core/api/services/guard"
	apitokens "github.com/cyril-fl/callback/core/api/tokens"
	"github.com/uptrace/bun"
)

// @title    callback API
// @version  1.0

// New gives the API, reachable by web pages from origin only (none when empty), with a valid token only.
func New(db bun.IDB, origin string) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /api/login", apilogin.Handler(db))
	mux.Handle("POST /api/logout", apilogin.Logout(db))
	mux.Handle("GET /api/me", apilogin.Me())

	mux.Handle("GET /api/applications", apireadapplications.Handler(db))
	mux.Handle("POST /api/applications", apicreateapplications.Handler(db))
	mux.Handle("PATCH /api/applications", apieditapplications.Handler(db))
	mux.Handle("DELETE /api/applications", apideleteapplications.Handler(db))
	mux.Handle("GET /api/stats", apireadstats.Handler(db))

	mux.Handle("GET /api/tokens", apitokens.List(db))
	mux.Handle("POST /api/tokens", apitokens.Create(db))
	mux.Handle("DELETE /api/tokens/{id}", apitokens.Delete(db))

	return guard.New(origin, auth.New(db, mux))
}
