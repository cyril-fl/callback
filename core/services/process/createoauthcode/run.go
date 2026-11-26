// Package createoauthcode is the consent of the OAuth page: the password checked, an authorization code for the client.
package createoauthcode

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/createsession"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// Lifetime is how long a code waits for its exchange: the client exchanges it at once.
const Lifetime = time.Minute

// Input is the consent of the OAuth page: the password, and what the client asked.
type Input struct {
	Password      string
	ClientID      string
	RedirectURI   string
	CodeChallenge string // S256 of the client's verifier
	ReadOnly      bool
}

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	input Input
}

type cache struct {
	userID string
	code   string
}

// Process checks the password and gives an authorization code for the client: the consent, without a session
// (createsession.ErrBadCredentials on a wrong password).
func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.checkPassword()
	p.saveCode()
}

// Getters & Setters

// GetResult is the code, for the client only: the base keeps its hash.
func (p *run) GetResult() string {
	return p.cache.code
}

// Methods
func (p *run) checkPassword() {
	userID, err := createsession.CheckPassword(p.ctx, p.db, p.input.Password)
	p.cache.userID = userID
	p.SetErr(err)
}

func (p *run) saveCode() {
	if p.HasErr() {
		return
	}
	code := password.Token()
	row := domain.OAuthCode{
		CodeHash:      password.TokenHash(code),
		UserID:        p.cache.userID,
		ClientID:      p.input.ClientID,
		RedirectURI:   p.input.RedirectURI,
		CodeChallenge: p.input.CodeChallenge,
		ReadOnly:      p.input.ReadOnly,
		// UTC: SQLite compares the times as text.
		ExpiresAt: time.Now().UTC().Add(Lifetime),
	}
	if _, err := p.db.NewInsert().Model(&row).Exec(p.ctx); err != nil {
		p.SetErr(err)
		return
	}
	p.cache.code = code
}
