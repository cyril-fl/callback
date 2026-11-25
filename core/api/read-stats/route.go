// Package readstats counts the applications: GET /api/stats.
package readstats

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/readstats"
	"github.com/uptrace/bun"
)

// Handler counts the applications by status and by source, and the share that got an answer, on a period or a source.
// @Summary  Count the applications
// @Tags     applications
// @Produce  json
// @Param    source          query string false "Source: spontaneous, job_post, recruiter or a name given by the user, any case"
// @Param    applied_before  query string false "applied_at < this day (2026-01-31)"
// @Param    applied_after   query string false "applied_at > this day (2026-01-31)"
// @Success  200 {object} readstats.Stats
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Router   /api/stats [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		process := readstats.Process(r.Context(), db, domain.Filter{
			Source:        domain.Source(query.Get("source")),
			AppliedBefore: query.Get("applied_before"),
			AppliedAfter:  query.Get("applied_after"),
		})
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
