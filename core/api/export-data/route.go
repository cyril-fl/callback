// Package exportdata gives everything the account wrote as one JSON file: GET /api/export.
package exportdata

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/exportdata"
	"github.com/uptrace/bun"
)

// Handler gives the applications, the trash, the status history and the follow-ups, to restore with POST /api/import.
// @Summary  Export all the data as JSON
// @Tags     data
// @Produce  json
// @Success  200 {object} domain.Backup
// @Failure  401 {object} respond.Failure
// @Router   /api/export [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := exportdata.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="callback-`+process.GetResult().ExportedAt.Format(domain.DateLayout)+`.json"`)
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
