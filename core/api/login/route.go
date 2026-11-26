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
	"github.com/cyril-fl/callback/core/services/process/editpassword"
	"github.com/uptrace/bun"
)

// Handler logs in: POST /api/login. The session token goes in an HttpOnly cookie, in the body only when the input
// asks it (the CLI). The multi mode needs the account's name.
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
func Handler(db bun.IDB, mode domain.ServerMode) http.HandlerFunc {
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

		process := createsession.Process(r.Context(), db, input, mode.Multi())
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

// MeResult is who asks: the kind of its token, whether it only reads, its account and the server's mode.
type MeResult struct {
	Kind     domain.TokenKind `json:"kind" enums:"session,api,oauth"`
	Name     string           `json:"name,omitempty" binding:"optional"` // the token's, for an API token or an OAuth connection
	User     string           `json:"user"`                              // the account's name
	Admin    bool             `json:"admin"`                             // the account manages the others, in the multi mode
	Mode     string           `json:"mode" enums:"single,multi"`
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
func Me(db bun.IDB, mode domain.ServerMode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := auth.Current(r.Context())
		var user domain.User
		if err := db.NewSelect().Model(&user).Column("name", "admin").Where("id = ?", domain.UserID(r.Context())).Scan(r.Context()); err != nil {
			respond.Error(w, r, err)
			return
		}
		respond.JSON(w, http.StatusOK, MeResult{Kind: token.Kind, Name: token.Name, User: user.Name, Admin: user.Admin, Mode: mode.Mode, ReadOnly: token.ReadOnly, Version: config.Version()})
	}
}

// EditPassword changes the password of the account logged in and logs its other devices out: PATCH /api/me/password.
// @Summary  Change the password of the account logged in
// @Tags     session
// @Accept   json
// @Param    body body editpassword.Input true "body"
// @Success  204
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  429 {object} respond.Failure
// @Router   /api/me/password [patch]
func EditPassword(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input editpassword.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		// By account, not address: a stolen session must not guess the password at leisure.
		key := "account " + domain.UserID(r.Context())
		failed := throttle.Passwords(db)
		if failed.Blocked(w, r, key) {
			return
		}

		process := editpassword.Process(r.Context(), db, input, auth.Current(r.Context()).ID)
		process.Run()
		if process.HasErr() {
			if !errors.Is(process.GetErr(), editpassword.ErrWrongPassword) {
				failed.Forgive(r.Context(), key)
			}
			respond.Error(w, r, process.GetErr())
			return
		}
		failed.Reset(r.Context(), key)
		w.WriteHeader(http.StatusNoContent)
	}
}
