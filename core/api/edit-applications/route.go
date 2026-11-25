// Package editapplications changes applications: PATCH /api/applications.
package editapplications

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/updateapplications"
	"github.com/uptrace/bun"
)

// Handler runs the ops in one transaction: each picks applications by ids or filters (never none: an empty one
// fails) and sets fields. An application the change makes invalid is listed in failed, the others are written.
// @Summary  Change applications
// @Tags     applications
// @Accept   json
// @Produce  json
// @Param    body body updateapplications.Input true "body"
// @Success  200 {object} domain.BatchResult
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/applications [patch]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input updateapplications.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		process := updateapplications.Process(r.Context(), db, input)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
