// Package restoreapplications restores deleted applications: POST /api/applications/restore.
package restoreapplications

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/restoreapplications"
	"github.com/uptrace/bun"
)

// Handler brings back applications still in the trash (trash_days), with their history. Ids not in the trash are
// listed in failed.
// @Summary  Restore deleted applications
// @Tags     applications
// @Accept   json
// @Produce  json
// @Param    body body restoreapplications.Input true "body"
// @Success  200 {object} domain.BatchResult
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/applications/restore [post]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input restoreapplications.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		process := restoreapplications.Process(r.Context(), db, input)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
