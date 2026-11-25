// Package login is the session: log in, log out, and who is asking.
package login

import (
	"errors"
	"net/http"
	"time"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/api/services/auth"
	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/api/services/throttle"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createsession"
	"github.com/cyril-fl/callback/core/services/process/deletesession"
	"github.com/uptrace/bun"
)

// Handler logs in: POST /api/login. The session token goes in an HttpOnly cookie, in the body only when the input
// asks it (the CLI).
// @Summary  Log in
// @Tags     session
// @Accept   json
// @Produce  json
// @Param    body body createsession.Input true "body"
// @Success  200 {object} createsession.Result
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  429 {object} respond.Failure
// @Router   /api/login [post]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input createsession.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		key := throttle.AddressKey(r)
		failed := throttle.Passwords(db)
		if failed.Blocked(w, r, key) {
			return
		}

		process := createsession.Process(r.Context(), db, input)
		process.Run()
		if process.HasErr() {
			if !errors.Is(process.GetErr(), createsession.ErrBadCredentials) {
				failed.Forgive(r.Context(), key)
			}
			respond.Error(w, r, process.GetErr())
			return
		}
		failed.Reset(r.Context(), key)
		result := process.GetResult()
		auth.SetCookie(w, result.Token, result.ExpiresAt)
		if !input.Token {
			result.Token = ""
		}
		respond.JSON(w, http.StatusOK, result)
	}
}

// Logout logs this device out: POST /api/logout.
// @Summary  Log out
// @Tags     session
// @Success  204
// @Failure  401 {object} respond.Failure
// @Router   /api/logout [post]
func Logout(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := deletesession.Process(r.Context(), db, auth.Token(r))
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		auth.SetCookie(w, "", time.Time{})
		w.WriteHeader(http.StatusNoContent)
	}
}

// MeResult is who asks: the kind of its token, and whether it only reads.
type MeResult struct {
	Kind     domain.TokenKind `json:"kind" enums:"session,api,oauth"`
	Name     string           `json:"name,omitempty" binding:"optional"`
	ReadOnly bool             `json:"read_only"`
	Version  string           `json:"version"`
}

// Me is who asks: GET /api/me.
// @Summary  Who is asking
// @Tags     session
// @Produce  json
// @Success  200 {object} login.MeResult
// @Failure  401 {object} respond.Failure
// @Router   /api/me [get]
func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := auth.Current(r.Context())
		respond.JSON(w, http.StatusOK, MeResult{Kind: token.Kind, Name: token.Name, ReadOnly: token.ReadOnly, Version: config.Version()})
	}
}
