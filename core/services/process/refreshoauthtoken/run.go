// Package refreshoauthtoken gives an OAuth connection new tokens for its refresh token.
package refreshoauthtoken

import (
	"context"
	"time"

	"github.com/cyril-fl/jat/core/domain"
	"github.com/cyril-fl/jat/core/services/password"
	"github.com/cyril-fl/jat/core/services/process/createoauthtoken"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// Input is a refresh (grant_type=refresh_token).
type Input struct {
	RefreshToken string
	ClientID     string
}

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	input Input
}

type cache struct {
	result createoauthtoken.Result
}

// Process gives an OAuth connection a new access token and a new refresh token, the old ones dead at once (rotation,
// as OAuth 2.1 asks for a public client), and 90 more days. A connection revoked, expired, or of another client, or a
// refresh token already used, is createoauthtoken.ErrInvalidGrant.
func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.rotateTokens()
}

// Getters & Setters
func (p *run) GetResult() createoauthtoken.Result {
	return p.cache.result
}

// Methods

// rotateTokens matches and replaces the refresh hash in one statement: of two refreshes at once, one wins.
func (p *run) rotateTokens() {
	result, row := createoauthtoken.NewTokens()
	update, err := p.db.NewUpdate().Model(&row).
		Column("hash", "refresh_hash", "access_expires_at", "expires_at").
		Where("refresh_hash = ? AND client_id = ? AND kind = ? AND expires_at > ?",
			password.TokenHash(p.input.RefreshToken), p.input.ClientID, domain.TokenOAuth, time.Now().UTC()).
		Exec(p.ctx)
	if err != nil {
		p.SetErr(err)
		return
	}
	if rows, _ := update.RowsAffected(); rows == 0 {
		p.SetErr(createoauthtoken.ErrInvalidGrant)
		return
	}
	p.cache.result = result
}
