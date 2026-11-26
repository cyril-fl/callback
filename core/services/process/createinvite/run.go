// Package createinvite makes an invitation for the sign-up: the base keeps its hash, and its token sealed
// when CALLBACK_SECRET is set.
package createinvite

import (
	"context"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/secret"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Result is the new invitation: Token goes in the sign-up link, and is shown once.
type Result struct {
	Token  string        `json:"token"`
	Invite domain.Invite `json:"invite"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx context.Context
	db  bun.IDB
}

type cache struct {
	result Result
}

// Process makes an invitation of the admin of ctx.
func Process(ctx context.Context, db bun.IDB) *run {
	return &run{
		ctx: ctx,
		db:  db,
	}
}

func (p *run) Run() {
	p.saveInvite()
}

// Getters & Setters
func (p *run) GetResult() Result {
	return p.cache.result
}

// Methods
func (p *run) saveInvite() {
	now := time.Now().UTC()
	p.cache.result.Token = password.Token()
	p.cache.result.Invite = domain.Invite{ID: uuid.NewString(), TokenHash: password.TokenHash(p.cache.result.Token), UserID: domain.UserID(p.ctx), CreatedAt: now, ExpiresAt: now.Add(domain.InviteLifetime)}
	invite := p.cache.result.Invite
	invite.Token, _ = secret.Seal(p.cache.result.Token) // without the secret, the link is shown once
	_, err := p.db.NewInsert().Model(&invite).Exec(p.ctx)
	p.cache.result.Invite.Token = p.cache.result.Token
	p.SetErr(err)
}
