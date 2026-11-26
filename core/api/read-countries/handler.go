// Package readcountries lists the countries an application can name: GET /api/countries.
package readcountries

import (
	"net/http"

	"github.com/cyril-fl/callback/core/api/services/respond"
	"github.com/cyril-fl/callback/core/domain"
)

// Handler gives the countries of countries.yaml: an application stores the code, takes the code or the English name.
// @Summary  List the countries
// @Tags     applications
// @Produce  json
// @Success  200 {array} domain.Country
// @Failure  401 {object} respond.Failure
// @Router   /api/countries [get]
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respond.List(w, http.StatusOK, domain.Countries)
	}
}
