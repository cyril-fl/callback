// Package setpassword sets the password of an account, and logs out its sessions: a new password is how a
// lost device is shut out. API tokens and OAuth connections stay: revoke them from the settings.
package setpassword

import (
	"context"
	"unicode/utf8"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// MinLength is the fewest characters of a password.
const MinLength = 8

var (
	ErrShortPassword = domain.Invalid("the password has at least 8 characters")
	ErrNoUser        = domain.NotFound("no such account")
)

type run struct {
	plumprimitive.Process

	ctx    context.Context
	db     bun.IDB
	userID string
	pass   string
	keep   string
}

func Process(ctx context.Context, db bun.IDB, userID, pass string) *run {
	return &run{
		ctx:    ctx,
		db:     db,
		userID: userID,
		pass:   pass,
	}
}

// Keep spares the session of id from the log out: the device that changed its own password.
func (p *run) Keep(id string) *run {
	p.keep = id
	return p
}

func (p *run) Run() {
	p.ensurePassword()
	p.savePassword()
}

// Methods
func (p *run) ensurePassword() {
	if utf8.RuneCountInString(p.pass) < MinLength {
		p.SetErr(ErrShortPassword)
	}
}

func (p *run) savePassword() {
	if p.HasErr() {
		return
	}
	hash := password.Hash(p.pass)
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().Model((*domain.User)(nil)).Set("password_hash = ?", hash).Where("id = ?", p.userID).Exec(ctx)
		if err != nil {
			return err
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			return ErrNoUser
		}
		_, err = tx.NewDelete().Model((*domain.Token)(nil)).Where("kind = ? AND user_id = ? AND id <> ?", domain.TokenSession, p.userID, p.keep).Exec(ctx)
		return err
	}))
}
