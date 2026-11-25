// Package readstatuschanges gives the status history: GET /api/status-changes.
package readstatuschanges

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/liststatuschanges"
	"github.com/uptrace/bun"
)

// Handler gives every status the applications reached and when, oldest first.
// @Summary  List the status history
// @Tags     applications
// @Produce  json
// @Success  200 {array} domain.StatusChange
// @Failure  401 {object} respond.Failure
// @Router   /api/status-changes [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := liststatuschanges.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.List(w, http.StatusOK, process.GetResult())
	}
}
