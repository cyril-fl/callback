// Package createsession logs in: the name and password checked, a session made for the device.
package createsession

import (
	"context"
	"strings"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Lifetime is how long a session lasts without use: each use starts it again (readtoken).
const Lifetime = 30 * 24 * time.Hour

var ErrBadCredentials = domain.Unauthorized("wrong name or password")

// dummyHash is checked when there is no account to check: the answer takes as long.
var dummyHash = password.Hash("not a password")

type Input struct {
	// Name is the account's, any case: required in the multi mode, optional in the single one, where it must be the
	// one account's.
	Name     string `json:"name,omitempty" binding:"optional"`
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
	multi bool
}

type cache struct {
	userID string
	result Result
}

// Process logs in to the account of input, multi telling the server's mode (domain.ServerMode.Multi).
func Process(ctx context.Context, db bun.IDB, input Input, multi bool) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
		multi: multi,
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
	userID, err := CheckPassword(p.ctx, p.db, p.input.Name, p.input.Password, p.multi)
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

// CheckPassword checks the name and password of an account and returns its id, in the same time whether the account
// exists or not: ErrBadCredentials either way, so the answer never tells which names exist. Without a name, the
// account is the base's only one, which the multi mode refuses. The OAuth consent checks it too, without a session.
func CheckPassword(ctx context.Context, db bun.IDB, name, pass string, multi bool) (string, error) {
	name = strings.TrimSpace(name)
	var users []domain.User
	query := db.NewSelect().Model(&users).Column("id", "password_hash").Limit(2)
	if name != "" {
		query = query.Where("lower(name) = lower(?)", name)
	}
	if err := query.Scan(ctx); err != nil {
		return "", err
	}
	if len(users) != 1 || multi && name == "" {
		password.Verify(pass, dummyHash)
		return "", ErrBadCredentials
	}
	if !password.Verify(pass, users[0].PasswordHash) {
		return "", ErrBadCredentials
	}
	return users[0].ID, nil
}
