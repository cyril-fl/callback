// Package api is the HTTP API: one folder per route, each running a process of core/services/process.
package api

import (
	"net/http"

	apicreateapplications "github.com/cyril-fl/callback/core/api/create-applications"
	apideleteapplications "github.com/cyril-fl/callback/core/api/delete-applications"
	apieditapplications "github.com/cyril-fl/callback/core/api/edit-applications"
	apiexportdata "github.com/cyril-fl/callback/core/api/export-data"
	apiforgetapplications "github.com/cyril-fl/callback/core/api/forget-applications"
	apiimportdata "github.com/cyril-fl/callback/core/api/import-data"
	apiinvites "github.com/cyril-fl/callback/core/api/invites"
	apilogin "github.com/cyril-fl/callback/core/api/login"
	apipreferences "github.com/cyril-fl/callback/core/api/preferences"
	apipurgeapplications "github.com/cyril-fl/callback/core/api/purge-applications"
	apireadapplications "github.com/cyril-fl/callback/core/api/read-applications"
	apireadcountries "github.com/cyril-fl/callback/core/api/read-countries"
	apireaddeletedapplications "github.com/cyril-fl/callback/core/api/read-deleted-applications"
	apireadsources "github.com/cyril-fl/callback/core/api/read-sources"
	apireadstats "github.com/cyril-fl/callback/core/api/read-stats"
	apireadstatuschanges "github.com/cyril-fl/callback/core/api/read-status-changes"
	apirestoreapplications "github.com/cyril-fl/callback/core/api/restore-applications"
	"github.com/cyril-fl/callback/core/api/services/auth"
	"github.com/cyril-fl/callback/core/api/services/guard"
	apitokens "github.com/cyril-fl/callback/core/api/tokens"
	apiusers "github.com/cyril-fl/callback/core/api/users"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/uptrace/bun"
)

// @title    callback API
// @version  1.0

// New gives the API, reachable by web pages from origin only (none when empty), with a valid token only, running in
// mode: the routes of the accounts exist in the multi mode only, 404 otherwise.
func New(db bun.IDB, origin string, mode domain.ServerMode) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /api/login", apilogin.Handler(db, mode))
	mux.Handle("POST /api/logout", apilogin.Logout(db))
	mux.Handle("GET /api/me", apilogin.Me(db, mode))
	mux.Handle("GET /api/add-user", apiusers.Modes(mode))
	if mode.Multi() {
		mux.Handle("POST /api/signup", apiusers.Signup(db, mode))
		mux.Handle("GET /api/users", auth.Admin(db, apiusers.List(db)))
		mux.Handle("DELETE /api/users/{id}", auth.Admin(db, apiusers.Delete(db)))
		mux.Handle("POST /api/users/{id}/password", auth.Admin(db, apiusers.ResetPassword(db)))
		mux.Handle("GET /api/invites", auth.Admin(db, apiinvites.List(db)))
		mux.Handle("POST /api/invites", auth.Admin(db, apiinvites.Create(db)))
		mux.Handle("DELETE /api/invites/{id}", auth.Admin(db, apiinvites.Delete(db)))
	}

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
	mux.Handle("GET /api/countries", apireadcountries.Handler())
	mux.Handle("GET /api/status-changes", apireadstatuschanges.Handler(db))

	mux.Handle("GET /api/export", apiexportdata.Handler(db))
	mux.Handle("POST /api/import", apiimportdata.Handler(db))

	mux.Handle("GET /api/preferences", apipreferences.Read(db))
	mux.Handle("PATCH /api/preferences", apipreferences.Update(db))

	mux.Handle("GET /api/tokens", apitokens.List(db))
	mux.Handle("POST /api/tokens", apitokens.Create(db))
	mux.Handle("DELETE /api/tokens/{id}", apitokens.Delete(db))

	return guard.New(origin, auth.New(db, mux))
}
