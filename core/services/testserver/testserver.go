// Package testserver serves the real API on a base of its own, for the tests of its clients (the routes, the CLI,
// the MCP, the OAuth).
package testserver

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/api"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/createapitoken"
	"github.com/cyril-fl/callback/core/services/process/createsession"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	"github.com/cyril-fl/callback/core/services/testdb"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Password is the password of the account Serve makes.
const Password = "correct horse"

// Server is the API under test.
type Server struct {
	DB  *bun.DB
	URL string
	// Session is the token of a session, as the CLI sends it.
	Session string
	// Ctx acts for the account of Session, for the processes a test runs on DB.
	Ctx context.Context
}

// Serve starts the API on a new base of one account whose password is Password, logged in once.
func Serve(t testing.TB) *Server {
	t.Helper()
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "admin")
	password := setpassword.Process(ctx, db, domain.UserID(ctx), Password)
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
	return &Server{DB: db, URL: server.URL, Session: session.GetResult().Token, Ctx: ctx}
}

// Token makes an API token of the account of Session, read-only or not.
func (s *Server) Token(t testing.TB, readOnly bool) string {
	t.Helper()
	process := createapitoken.Process(s.Ctx, s.DB, createapitoken.Input{Name: t.Name(), ReadOnly: readOnly})
	process.Run()
	if process.HasErr() {
		t.Fatal(process.GetErr())
	}
	return process.GetResult().Token
}

// Account adds the account name to the base and returns the token of a session of it: the login cannot tell two
// accounts apart by their password alone.
func (s *Server) Account(t testing.TB, name string) string {
	t.Helper()
	ctx := testdb.Account(t, s.DB, name)
	now := time.Now().UTC()
	token := password.Token()
	session := domain.Token{ID: uuid.NewString(), UserID: domain.UserID(ctx), Kind: domain.TokenSession, Hash: password.TokenHash(token), CreatedAt: now, ExpiresAt: now.Add(createsession.Lifetime)}
	if _, err := s.DB.NewInsert().Model(&session).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	return token
}
