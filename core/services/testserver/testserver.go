// Package testserver serves the real API on a base of its own, for the tests of its clients (the routes, the CLI,
// the MCP, the OAuth).
package testserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cyril-fl/jat/core/api"
	"github.com/cyril-fl/jat/core/services/process/createapitoken"
	"github.com/cyril-fl/jat/core/services/process/createsession"
	"github.com/cyril-fl/jat/core/services/process/setpassword"
	"github.com/cyril-fl/jat/core/services/testdb"
	"github.com/uptrace/bun"
)

// Password is the password of the base Serve makes.
const Password = "correct horse"

// Server is the API under test.
type Server struct {
	DB  *bun.DB
	URL string
	// Session is the token of a session, as the CLI sends it.
	Session string
}

// Serve starts the API on a new base whose password is Password, logged in once.
func Serve(t testing.TB) *Server {
	t.Helper()
	db := testdb.New(t)
	ctx := context.Background()
	password := setpassword.Process(ctx, db, Password)
	password.Run()
	if password.HasErr() {
		t.Fatal(password.GetErr())
	}
	session := createsession.Process(ctx, db, createsession.Input{Password: Password})
	session.Run()
	if session.HasErr() {
		t.Fatal(session.GetErr())
	}
	server := httptest.NewServer(api.New(db, ""))
	t.Cleanup(server.Close)
	return &Server{DB: db, URL: server.URL, Session: session.GetResult().Token}
}

// Token makes an API token, read-only or not.
func (s *Server) Token(t testing.TB, readOnly bool) string {
	t.Helper()
	process := createapitoken.Process(context.Background(), s.DB, createapitoken.Input{Name: t.Name(), ReadOnly: readOnly})
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	return process.GetResult().Token
}
