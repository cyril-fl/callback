// Package createapitoken makes an API token, for a script or an agent: shown once, the base keeps its hash.
package createapitoken

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Prefix starts every API token: a leaked one is recognizable.
const Prefix = "callback_"

// Lifetimes are the durations a token can be given; 0 is no limit.
var Lifetimes = map[string]time.Duration{
	"7d":    7 * 24 * time.Hour,
	"30d":   30 * 24 * time.Hour,
	"90d":   90 * 24 * time.Hour,
	"1y":    365 * 24 * time.Hour,
	"never": 0,
}

// DefaultLifetime is the duration of a token when none is asked.
const DefaultLifetime = "90d"

// MaxName is the most characters of a token's name.
const MaxName = 100

var (
	ErrNoName          = domain.Invalid("the token needs a name")
	ErrLongName        = domain.Invalid("the name of the token has at most 100 characters")
	ErrInvalidLifetime = domain.Invalid("expires_in must be one of 7d, 30d, 90d, 1y, never")
)

type Input struct {
	Name string `json:"name"`
	// ExpiresIn is how long the token works, 90d when left out.
	ExpiresIn string `json:"expires_in,omitempty" binding:"optional" enums:"7d,30d,90d,1y,never"`
	// ReadOnly makes a token that reads and changes nothing.
	ReadOnly bool `json:"read_only,omitempty" binding:"optional"`
}

// Result is the new token: Token is shown once.
type Result struct {
	Token    string       `json:"token"`
	APIToken domain.Token `json:"api_token"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	input Input
}

type cache struct {
	lifetime time.Duration
	result   Result
}

func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.ensureInput()
	p.saveToken()
}

// Getters & Setters
func (p *run) GetResult() Result {
	return p.cache.result
}

// Methods
func (p *run) ensureInput() {
	p.input.Name = strings.TrimSpace(p.input.Name)
	switch {
	case p.input.Name == "":
		p.SetErr(ErrNoName)
		return
	case utf8.RuneCountInString(p.input.Name) > MaxName:
		p.SetErr(ErrLongName)
		return
	}
	if p.input.ExpiresIn == "" {
		p.input.ExpiresIn = DefaultLifetime
	}
	lifetime, ok := Lifetimes[p.input.ExpiresIn]
	if !ok {
		p.SetErr(ErrInvalidLifetime)
		return
	}
	p.cache.lifetime = lifetime
}

func (p *run) saveToken() {
	if p.HasErr() {
		return
	}
	now := time.Now().UTC()
	p.cache.result.Token = Prefix + password.Token()
	token := &p.cache.result.APIToken
	*token = domain.Token{ID: uuid.NewString(), Kind: domain.TokenAPI, Name: p.input.Name, Hash: password.TokenHash(p.cache.result.Token), ReadOnly: p.input.ReadOnly, CreatedAt: now}
	if p.cache.lifetime > 0 {
		token.ExpiresAt = now.Add(p.cache.lifetime)
	}
	_, err := p.db.NewInsert().Model(token).Exec(p.ctx)
	p.SetErr(err)
}
