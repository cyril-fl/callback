// Package forgetapplications deletes applications from the trash for good: DELETE /api/applications/deleted.
package forgetapplications

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/forgetapplications"
	"github.com/uptrace/bun"
)

// Handler deletes for good, with their history, the applications of ids in the trash, or all of them with all. A
// session only: an API or OAuth token is refused. Ids not in the trash are listed in failed.
// @Summary  Delete applications from the trash for good
// @Tags     applications
// @Accept   json
// @Produce  json
// @Param    body body forgetapplications.Input true "body"
// @Success  200 {object} domain.BatchResult
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/applications/deleted [delete]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input forgetapplications.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		process := forgetapplications.Process(r.Context(), db, input)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
