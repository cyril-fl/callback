// Package importdata restores an export: POST /api/import.
package importdata

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/importdata"
	"github.com/uptrace/bun"
)

// Handler adds what the export holds and the base lacks, in one transaction. An application the base already has
// (live or in the trash) is left as it is, so the same file can be imported twice; the invalid ones are in failed.
// @Summary  Restore an export
// @Tags     data
// @Accept   json
// @Produce  json
// @Param    backup body domain.Backup true "The file GET /api/export gave"
// @Success  200 {object} importdata.Result
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Router   /api/import [post]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var backup domain.Backup
		if !respond.Decode(w, r, &backup) {
			return
		}
		process := importdata.Process(r.Context(), db, backup)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
