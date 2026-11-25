// Package createapplications creates applications: POST /api/applications.
package createapplications

import (
	"net/http"

	"github.com/cyril-fl/jat/core/api/services/respond"
	"github.com/cyril-fl/jat/core/domain"
	"github.com/cyril-fl/jat/core/services/process/createapplications"
	"github.com/uptrace/bun"
)

// Handler creates a batch of applications in one transaction: the valid ones are written, the others listed in
// failed by index. 201 when at least one was created.
// @Summary  Create applications
// @Tags     applications
// @Accept   json
// @Produce  json
// @Param    body body []domain.NewApplication true "At most 500 applications"
// @Success  201 {object} domain.BatchResult
// @Success  200 {object} domain.BatchResult "None created: every item failed"
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/applications [post]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var items []domain.NewApplication
		if !respond.Decode(w, r, &items) {
			return
		}
		process := createapplications.Process(r.Context(), db, items)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		status := http.StatusCreated
		if process.GetResult().Count == 0 {
			status = http.StatusOK
		}
		respond.JSON(w, status, process.GetResult())
	}
}
