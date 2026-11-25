// Package readtoken finds the token a client sent: a session, which starts its lifetime again, an API token, or the
// access token of an OAuth connection.
package readtoken

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/createsession"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var ErrNoSession = domain.Unauthorized("not logged in, or the token expired")

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	token string
}

type cache struct {
	result domain.Token
}

func Process(ctx context.Context, db bun.IDB, token string) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		token: token,
	}
}

func (p *run) Run() {
	p.loadToken()
	p.touchToken()
}

// Getters & Setters
func (p *run) GetResult() domain.Token {
	return p.cache.result
}

// Methods
func (p *run) loadToken() {
	if p.token == "" {
		p.SetErr(ErrNoSession)
		return
	}
	now := time.Now().UTC()
	err := p.db.NewSelect().Model(&p.cache.result).
		Where("hash = ?", password.TokenHash(p.token)).
		Where("kind != ? OR access_expires_at > ?", domain.TokenOAuth, now).
		Where("expires_at IS NULL OR expires_at > ?", now).
		Scan(p.ctx)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNoSession
	}
	p.SetErr(err)
}

// touchToken records the use, and keeps a session in use alive.
func (p *run) touchToken() {
	if p.HasErr() {
		return
	}
	token := &p.cache.result
	token.LastUsedAt = time.Now().UTC()
	query := p.db.NewUpdate().Model(token).Column("last_used_at").WherePK()
	if token.Kind == domain.TokenSession {
		token.ExpiresAt = token.LastUsedAt.Add(createsession.Lifetime)
		query = query.Column("expires_at")
	}
	_, err := query.Exec(p.ctx)
	p.SetErr(err)
}
