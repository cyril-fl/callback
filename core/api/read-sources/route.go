// Package readsources lists the sources in use: GET /api/sources.
package readsources

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/listsources"
	"github.com/uptrace/bun"
)

// Handler gives the sources the applications use, the most used first, for the form to offer them.
// @Summary  List the sources in use
// @Tags     applications
// @Produce  json
// @Success  200 {array} listsources.Source
// @Failure  401 {object} respond.Failure
// @Router   /api/sources [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := listsources.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.List(w, http.StatusOK, process.GetResult())
	}
}
