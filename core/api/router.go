// Package api is the HTTP API: one folder per route, each running a process of core/services/process.
package api

import (
	"net/http"

	apicreateapplications "github.com/cyril-fl/callback/core/api/create-applications"
	apideleteapplications "github.com/cyril-fl/callback/core/api/delete-applications"
	apieditapplications "github.com/cyril-fl/callback/core/api/edit-applications"
	apiforgetapplications "github.com/cyril-fl/callback/core/api/forget-applications"
	apilogin "github.com/cyril-fl/callback/core/api/login"
	apipreferences "github.com/cyril-fl/callback/core/api/preferences"
	apipurgeapplications "github.com/cyril-fl/callback/core/api/purge-applications"
	apireadapplications "github.com/cyril-fl/callback/core/api/read-applications"
	apireaddeletedapplications "github.com/cyril-fl/callback/core/api/read-deleted-applications"
	apireadsources "github.com/cyril-fl/callback/core/api/read-sources"
	apireadstats "github.com/cyril-fl/callback/core/api/read-stats"
	apireadstatuschanges "github.com/cyril-fl/callback/core/api/read-status-changes"
	apirestoreapplications "github.com/cyril-fl/callback/core/api/restore-applications"
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
	mux.Handle("GET /api/applications/deleted", apireaddeletedapplications.Handler(db))
	mux.Handle("DELETE /api/applications/deleted", apiforgetapplications.Handler(db))
	mux.Handle("POST /api/applications/restore", apirestoreapplications.Handler(db))
	mux.Handle("DELETE /api/data", apipurgeapplications.Handler(db))
	mux.Handle("GET /api/stats", apireadstats.Handler(db))
	mux.Handle("GET /api/sources", apireadsources.Handler(db))
	mux.Handle("GET /api/status-changes", apireadstatuschanges.Handler(db))

	mux.Handle("GET /api/preferences", apipreferences.Read(db))
	mux.Handle("PATCH /api/preferences", apipreferences.Update(db))

	mux.Handle("GET /api/tokens", apitokens.List(db))
	mux.Handle("POST /api/tokens", apitokens.Create(db))
	mux.Handle("DELETE /api/tokens/{id}", apitokens.Delete(db))

	return guard.New(origin, auth.New(db, mux))
}
