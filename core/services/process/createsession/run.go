// Package createsession logs in: the password checked, a session made for the device.
package createsession

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Lifetime is how long a session lasts without use: each use starts it again (readtoken).
const Lifetime = 30 * 24 * time.Hour

var ErrBadCredentials = domain.Unauthorized("wrong password")

// dummyHash is checked when there is no account to check: the answer takes as long.
var dummyHash = password.Hash("not a password")

type Input struct {
	Password string `json:"password"`
	// Token asks the token in the answer too, for a client without cookies (the CLI): the web keeps it HttpOnly.
	Token bool `json:"token,omitempty" binding:"optional"`
}

// Result is the new session: the token goes to the client, and only there.
type Result struct {
	Token     string    `json:"token,omitempty" binding:"optional"`
	ExpiresAt time.Time `json:"expires_at"`
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
	result Result
}

func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.checkPassword()
	p.saveSession()
}

// Getters & Setters
func (p *run) GetResult() Result {
	return p.cache.result
}

// Methods
func (p *run) checkPassword() {
	userID, err := CheckPassword(p.ctx, p.db, p.input.Password)
	p.cache.userID = userID
	p.SetErr(err)
}

func (p *run) saveSession() {
	if p.HasErr() {
		return
	}
	now := time.Now().UTC()
	p.cache.result.Token = password.Token()
	p.cache.result.ExpiresAt = now.Add(Lifetime)
	session := domain.Token{ID: uuid.NewString(), UserID: p.cache.userID, Kind: domain.TokenSession, Hash: password.TokenHash(p.cache.result.Token), CreatedAt: now, LastUsedAt: now, ExpiresAt: p.cache.result.ExpiresAt}
	_, err := p.db.NewInsert().Model(&session).Exec(p.ctx)
	p.SetErr(err)
}

// Helpers

// CheckPassword checks the password of the base's account and returns its id, in the same time whether there is one
// or not: ErrBadCredentials either way. The OAuth consent checks it too, without a session.
// shortcut: the password alone names the account, so a base of several accounts logs nobody in; the login takes a
// name with the multi-user mode.
func CheckPassword(ctx context.Context, db bun.IDB, pass string) (string, error) {
	var users []domain.User
	if err := db.NewSelect().Model(&users).Column("id", "password_hash").Limit(2).Scan(ctx); err != nil {
		return "", err
	}
	if len(users) != 1 {
		password.Verify(pass, dummyHash)
		return "", ErrBadCredentials
	}
	if !password.Verify(pass, users[0].PasswordHash) {
		return "", ErrBadCredentials
	}
	return users[0].ID, nil
}
