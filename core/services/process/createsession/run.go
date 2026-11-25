// Package createsession logs in: the password checked, a session made for the device.
package createsession

import (
	"context"
	"database/sql"
	"errors"
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

// dummyHash is checked when no password is set: the answer takes as long.
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
	p.SetErr(CheckPassword(p.ctx, p.db, p.input.Password))
}

func (p *run) saveSession() {
	if p.HasErr() {
		return
	}
	now := time.Now().UTC()
	p.cache.result.Token = password.Token()
	p.cache.result.ExpiresAt = now.Add(Lifetime)
	session := domain.Token{ID: uuid.NewString(), Kind: domain.TokenSession, Hash: password.TokenHash(p.cache.result.Token), CreatedAt: now, LastUsedAt: now, ExpiresAt: p.cache.result.ExpiresAt}
	_, err := p.db.NewInsert().Model(&session).Exec(p.ctx)
	p.SetErr(err)
}

// Helpers

// CheckPassword checks the password, in the same time whether one is set or not: ErrBadCredentials either way.
// The OAuth consent checks it too, without a session.
func CheckPassword(ctx context.Context, db bun.IDB, pass string) error {
	var setting domain.Setting
	err := db.NewSelect().Model(&setting).Where("key = ?", domain.SettingPasswordHash).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		password.Verify(pass, dummyHash)
		return ErrBadCredentials
	}
	if err != nil {
		return err
	}
	if !password.Verify(pass, setting.Value) {
		return ErrBadCredentials
	}
	return nil
}
