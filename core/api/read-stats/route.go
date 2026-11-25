// Package readstats counts the applications: GET /api/stats.
package readstats

import (
	"net/http"

	"github.com/cyril-fl/jat/core/api/services/respond"
	"github.com/cyril-fl/jat/core/services/process/readstats"
	"github.com/uptrace/bun"
)

// Handler counts the applications by status, and the share that got an answer.
// @Summary  Count the applications
// @Tags     applications
// @Produce  json
// @Success  200 {object} readstats.Stats
// @Failure  401 {object} respond.Failure
// @Router   /api/stats [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := readstats.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
