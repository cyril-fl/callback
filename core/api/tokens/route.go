// Package tokens manages the API tokens and OAuth connections: refused to a token, a login only.
package tokens

import (
	"net/http"

	"github.com/cyril-fl/jat/core/api/services/respond"
	"github.com/cyril-fl/jat/core/services/process/createapitoken"
	"github.com/cyril-fl/jat/core/services/process/deletetoken"
	"github.com/cyril-fl/jat/core/services/process/listtokens"
	"github.com/uptrace/bun"
)

// List gives the API tokens and OAuth connections, newest first.
// @Summary  List the tokens
// @Tags     tokens
// @Produce  json
// @Success  200 {array} domain.Token
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/tokens [get]
func List(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := listtokens.Process(r.Context(), db)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.List(w, http.StatusOK, process.GetResult())
	}
}

// Create makes an API token, shown once.
// @Summary  Create an API token
// @Tags     tokens
// @Accept   json
// @Produce  json
// @Param    body body createapitoken.Input true "body"
// @Success  201 {object} createapitoken.Result
// @Failure  400 {object} respond.Failure
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Router   /api/tokens [post]
func Create(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input createapitoken.Input
		if !respond.Decode(w, r, &input) {
			return
		}
		process := createapitoken.Process(r.Context(), db, input)
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		respond.JSON(w, http.StatusCreated, process.GetResult())
	}
}

// Delete revokes an API token or an OAuth connection.
// @Summary  Revoke a token
// @Tags     tokens
// @Param    id path string true "Token id"
// @Success  204
// @Failure  401 {object} respond.Failure
// @Failure  403 {object} respond.Failure
// @Failure  404 {object} respond.Failure
// @Router   /api/tokens/{id} [delete]
func Delete(db bun.IDB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process := deletetoken.Process(r.Context(), db, r.PathValue("id"))
		process.Run()
		if process.HasErr() {
			respond.Error(w, r, process.GetErr())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
