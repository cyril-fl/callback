// Package invites is the invitations to sign up, which an admin hands out in the invitation mode.
package invites

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/createinvite"
	"github.com/cyril-fl/callback/core/services/process/deleteinvite"
	"github.com/cyril-fl/callback/core/services/process/listinvites"
	"github.com/uptrace/bun"
)

// List gives the invitations still usable, of every admin, newest first.
// @Summary  List the invitations
// @Tags     invites
// @Produce  json
// @Success  200 {array} domain.Invite
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Router   /api/invites [get]
func List(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := listinvites.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.List(w, http.StatusOK, process.GetResult())
	}
}

// Create makes an invitation for 7 days: its token, for the sign-up, is shown once.
// @Summary  Create an invitation
// @Tags     invites
// @Produce  json
// @Success  201 {object} createinvite.Result
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Router   /api/invites [post]
func Create(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := createinvite.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusCreated, process.GetResult())
	}
}

// Delete revokes an invitation.
// @Summary  Revoke an invitation
// @Tags     invites
// @Param    id path string true "Invitation id"
// @Success  204
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Router   /api/invites/{id} [delete]
func Delete(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := deleteinvite.Process(r.Context(), db, r.PathValue("id"))
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
