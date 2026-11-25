// Package readapplications lists applications: GET /api/applications.
package readapplications

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cyril-fl/jat/core/api/services/respond"
	"github.com/cyril-fl/jat/core/domain"
	"github.com/cyril-fl/jat/core/services/process/listapplications"
	"github.com/uptrace/bun"
)

// Handler lists the applications that match the filters, one page of them; X-Total-Count is the count of all.
// @Summary  List applications
// @Tags     applications
// @Produce  json
// @Param    ids             query []string false "Applications by id, comma separated or repeated" collectionFormat(csv)
// @Param    status          query []string false "Any of these statuses" collectionFormat(csv) Enums(applied,responded,interview,offer,rejected)
// @Param    source          query string   false "Source" Enums(spontaneous,job_post,recruiter)
// @Param    q               query string   false "Text in company, location, website, source_detail or comment"
// @Param    applied_before  query string   false "applied_at < this day (2026-01-31)"
// @Param    applied_after   query string   false "applied_at > this day (2026-01-31)"
// @Param    untouched_since query string   false "Not updated since this day (2026-01-31)"
// @Param    fields          query []string false "Columns returned, all when empty; id always comes" collectionFormat(csv)
// @Param    max_chars       query int      false "Cut each comment to this many characters (truncated: true)"
// @Param    sort            query string   false "applied_at, company, source, status, rating, created_at, updated_at; - first to reverse; -applied_at by default"
// @Param    limit           query int      false "Most applications returned, all when 0"
// @Param    offset          query int      false "Applications skipped"
// @Param    count_only      query bool     false "No application, X-Total-Count only"
// @Success  200 {array} domain.Application
// @Header   200 {integer} X-Total-Count "Count of all the applications that match"
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Router   /api/applications [get]
func Handler(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input, err := parse(r.URL.Query())
		if err != nil {
			respond.Fail(w, http.StatusBadRequest, err.Error())
			return
		}
		process := listapplications.Process(r.Context(), db, input)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.Total(w, process.GetResult().Total)
		respond.List(w, http.StatusOK, process.GetResult().Applications)
	}
}

var errNumber = domain.Invalid("limit, offset and max_chars are numbers, count_only true or false")

func parse(query url.Values) (listapplications.Input, error) {
	input := listapplications.Input{
		Filter: domain.Filter{
			IDs:            list(query["ids"]),
			Source:         domain.Source(query.Get("source")),
			Q:              query.Get("q"),
			AppliedBefore:  query.Get("applied_before"),
			AppliedAfter:   query.Get("applied_after"),
			UntouchedSince: query.Get("untouched_since"),
		},
		Fields: list(query["fields"]),
		Sort:   query.Get("sort"),
	}
	for _, status := range list(query["status"]) {
		input.Status = append(input.Status, domain.Status(status))
	}
	var err error
	for name, dest := range map[string]*int{"limit": &input.Limit, "offset": &input.Offset, "max_chars": &input.MaxChars} {
		if value := query.Get(name); value != "" {
			if *dest, err = strconv.Atoi(value); err != nil {
				return input, errNumber
			}
		}
	}
	if value := query.Get("count_only"); value != "" {
		if input.CountOnly, err = strconv.ParseBool(value); err != nil {
			return input, errNumber
		}
	}
	return input, nil
}

// list reads a parameter given comma separated, repeated, or both.
func list(values []string) []string {
	var result []string
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			if item = strings.TrimSpace(item); item != "" {
				result = append(result, item)
			}
		}
	}
	return result
}
