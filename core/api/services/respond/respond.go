// Package respond is what every route shares: reading the request, writing JSON, turning process errors into statuses.
package respond

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/cyril-fl/callback/core/domain"
)

// maxBody limits a request body: a batch of applications is far below it.
const maxBody = 10 << 20

var statuses = []struct {
	status int
	errs   []error
}{
	{http.StatusBadRequest, []error{domain.ErrInvalid}},
	{http.StatusUnauthorized, []error{domain.ErrUnauthorized}},
	{http.StatusForbidden, []error{domain.ErrForbidden}},
	{http.StatusNotFound, []error{domain.ErrNotFound}},
}

// JSON writes data with this status, compact: agents pay for every byte.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("write response: %v", err)
	}
}

// List writes a list: an empty one is [], not null.
func List[T any](w http.ResponseWriter, status int, items []T) {
	JSON(w, status, append([]T{}, items...))
}

// Total counts in X-Total-Count the whole list a page comes from. Before the body.
func Total(w http.ResponseWriter, total int) {
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
}

// Error writes the status of a process error. An unknown error is a bug or an outage: its detail may expose SQL, so
// it only goes to the logs.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	for _, group := range statuses {
		for _, known := range group.errs {
			if errors.Is(err, known) {
				Fail(w, group.status, err.Error())
				return
			}
		}
	}

	log.Printf("internal error: %s %s: %v", r.Method, r.URL.Path, err)
	Fail(w, http.StatusInternalServerError, "internal error")
}

// Decode reads the JSON body into dest; on false the response is already written.
func Decode(w http.ResponseWriter, r *http.Request, dest any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dest)
	var tooLarge *http.MaxBytesError
	switch {
	case err == nil:
		return true
	case errors.As(err, &tooLarge):
		Fail(w, http.StatusRequestEntityTooLarge, "request body too large")
	default:
		Fail(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
	}
	return false
}

// Failure is the body of every error.
type Failure struct {
	Error string `json:"error"`
}

// Fail writes an error with this status.
func Fail(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Failure{Error: message})
}
