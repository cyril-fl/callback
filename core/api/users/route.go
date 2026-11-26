// Package users is the accounts of the multi mode: the sign-up, and the routes an admin runs the others with.
package users

import (
	"errors"
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/auth"
	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/api/services/throttle"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/createsession"
	"github.com/cyril-fl/callback/core/services/process/createuser"
	"github.com/cyril-fl/callback/core/services/process/deleteuser"
	"github.com/cyril-fl/callback/core/services/process/listusers"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	"github.com/uptrace/bun"
)

// Reset is the new password of an account: it goes to the admin once, the base keeping only its hash.
type Reset struct {
	Password string `json:"password"`
}

// Modes tells the login and sign-up pages how the server runs, without a session: GET /api/add-user.
// @Summary  How the server runs: its mode and how accounts are created
// @Tags     users
// @Produce  json
// @Success  200 {object} domain.ServerMode
// @Router   /api/add-user [get]
func Modes(mode domain.ServerMode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, mode)
	}
}

// Signup creates an account and logs it in, without a session: POST /api/signup, in the multi mode only. It needs an
// invitation in the invitation mode, but for the first account of a base, which becomes admin. Its tries count with
// the wrong passwords of the address (throttle.Passwords): an invitation cannot be guessed either. In the public mode
// the accounts of an address count too (throttle.Signups): a few, then it waits longer at each one.
// @Summary  Create an account and log it in
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    body body createuser.Input true "body"
// @Success  200 {object} createsession.Result
// @Failure  400 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Failure  429 {object} respond.Failure
// @Router   /api/signup [post]
func Signup(db bun.IDB, mode domain.ServerMode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input createuser.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		key := throttle.AddressKey(r)
		// created counts the accounts of the address in the public mode, where no invitation holds the sign-up back.
		created := throttle.Signups(db)
		public := mode.Public()
		if public && created.Blocked(w, r, key) {
			return
		}
		failed := throttle.Passwords(db)
		if failed.Blocked(w, r, key) {
			if public {
				created.Forgive(r.Context(), key)
			}
			return
		}

		process := createuser.Process(r.Context(), db, input, public)
		process.Run()
		if process.HasErr() {
			if public {
				created.Forgive(r.Context(), key)
			}
			if !errors.Is(process.GetErr(), createuser.ErrBadInvite) {
				failed.Forgive(r.Context(), key)
			}
			respond.Error(w, r, process.GetErr())
			return
		}
		failed.Reset(r.Context(), key)

		session := createsession.Process(r.Context(), db, createsession.Input{Name: input.Name, Password: input.Password}, true)
		session.Run()
		if session.HasErr() {
			respond.Error(w, r, session.GetErr())
			return
		}
		result := session.GetResult()
		auth.SetCookie(w, result.Token, result.ExpiresAt)
		result.Token = ""
		respond.JSON(w, http.StatusOK, result)
	}
}

// List gives every account, by name: GET /api/users.
// @Summary  List the accounts
// @Tags     users
// @Produce  json
// @Success  200 {array} domain.User
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Router   /api/users [get]
func List(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := listusers.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.List(w, http.StatusOK, process.GetResult())
	}
}

// Delete deletes an account for good, with all its data: DELETE /api/users/{id}.
// @Summary  Delete an account and all its data
// @Tags     users
// @Param    id path string true "Account id"
// @Success  204
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Router   /api/users/{id} [delete]
func Delete(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := deleteuser.Process(r.Context(), db, r.PathValue("id"))
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ResetPassword gives an account a random password, in the answer only, and logs its sessions out:
// POST /api/users/{id}/password.
// @Summary  Reset the password of an account
// @Tags     users
// @Produce  json
// @Param    id path string true "Account id"
// @Success  200 {object} users.Reset
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Router   /api/users/{id}/password [post]
func ResetPassword(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reset := Reset{Password: password.Token()}
		process := setpassword.Process(r.Context(), db, r.PathValue("id"), reset.Password)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, reset)
	}
}
