// Package purgeapplications deletes every application, to start a new job search: DELETE /api/data.
package purgeapplications

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/purgeapplications"
	"github.com/uptrace/bun"
)

// Result is how many applications were deleted.
type Result struct {
	Count int `json:"count"`
}

// Handler deletes every application and its status history, for good; the settings and the tokens stay. A session
// only: an API or OAuth token, an agent's, cannot.
// @Summary  Delete every application
// @Tags     applications
// @Produce  json
// @Success  200 {object} purgeapplications.Result
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/data [delete]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := purgeapplications.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, Result{Count: process.GetResult()})
	}
}
