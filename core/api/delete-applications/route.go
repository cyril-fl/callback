// Package deleteapplications deletes applications: DELETE /api/applications.
package deleteapplications

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/deleteapplications"
	"github.com/uptrace/bun"
)

// Handler deletes the applications of ids or filters; an empty filter is refused. Ids that match nothing are
// listed in failed.
// @Summary  Delete applications
// @Tags     applications
// @Accept   json
// @Produce  json
// @Param    body body deleteapplications.Input true "body"
// @Success  200 {object} domain.BatchResult
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/applications [delete]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input deleteapplications.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		process := deleteapplications.Process(r.Context(), db, input)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
