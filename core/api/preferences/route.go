// Package preferences reads and changes the preferences of the account: GET and PATCH /api/preferences.
package preferences

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/readpreferences"
	"github.com/cyril-fl/callback/core/services/process/updatepreferences"
	"github.com/uptrace/bun"
)

// Read gives the preferences, the defaults where none was saved.
// @Summary  Read the preferences
// @Tags     preferences
// @Produce  json
// @Success  200 {object} domain.Preferences
// @Failure  401 {object} respond.Failure
// @Router   /api/preferences [get]
func Read(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := readpreferences.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}

// Update sets the preferences given, the others keep their value.
// @Summary  Change the preferences
// @Tags     preferences
// @Accept   json
// @Produce  json
// @Param    body body domain.PreferenceChanges true "body"
// @Success  200 {object} domain.Preferences
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Router   /api/preferences [patch]
func Update(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var changes domain.PreferenceChanges
		if !respond.Decode(w, r, &changes) {
			return
		}
		process := updatepreferences.Process(r.Context(), db, changes)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
