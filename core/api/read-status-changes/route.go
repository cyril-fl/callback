// Package readstatuschanges gives the status history: GET /api/status-changes.
package readstatuschanges

import (
	"net/http"
	"strings"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/services/process/liststatuschanges"
	"github.com/uptrace/bun"
)

// Handler gives the statuses the applications reached and when, oldest first; all of them without a filter.
// @Summary  List the status history
// @Tags     applications
// @Produce  json
// @Param    ids   query []string false "Of these applications, comma separated or repeated" collectionFormat(csv)
// @Param    since query string   false "Reached on this day or after (2026-01-31)"
// @Success  200 {array} domain.StatusChange
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Router   /api/status-changes [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := liststatuschanges.Process(r.Context(), db, liststatuschanges.Input{
			IDs:   ids(r.URL.Query()["ids"]),
			Since: r.URL.Query().Get("since"),
		})
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.List(w, http.StatusOK, process.GetResult())
	}
}

// ids reads the ids given comma separated, repeated, or both.
func ids(values []string) []string {
	var result []string
	for _, value := range values {
		for _, id := range strings.Split(value, ",") {
			if id = strings.TrimSpace(id); id != "" {
				result = append(result, id)
			}
		}
	}
	return result
}
