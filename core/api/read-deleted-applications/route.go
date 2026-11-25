// Package readdeletedapplications lists the trash: GET /api/applications/deleted.
package readdeletedapplications

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/listdeletedapplications"
	"github.com/uptrace/bun"
)

// Handler lists the applications deleted within the preferences' trash_days, the last deleted first; POST
// /api/applications/restore brings them back.
// @Summary  List deleted applications
// @Tags     applications
// @Produce  json
// @Success  200 {array} domain.DeletedApplication
// @Failure  401 {object} respond.Failure
// @Router   /api/applications/deleted [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := listdeletedapplications.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusOK, process.GetResult())
	}
}
