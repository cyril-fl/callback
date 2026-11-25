// Package setpassword sets the password of the one account, and logs out every session: a new password is how a
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

var ErrShortPassword = domain.Invalid("the password has at least 8 characters")

type run struct {
	plumprimitive.Process

	ctx  context.Context
	db   bun.IDB
	pass string
}

func Process(ctx context.Context, db bun.IDB, pass string) *run {
	return &run{
		ctx:  ctx,
		db:   db,
		pass: pass,
	}
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
	setting := domain.Setting{Key: domain.SettingPasswordHash, Value: password.Hash(p.pass)}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&setting).On("CONFLICT (key) DO UPDATE").Set("value = EXCLUDED.value").Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*domain.Token)(nil)).Where("kind = ?", domain.TokenSession).Exec(ctx)
		return err
	}))
}

// HasPassword tells whether the base has a password yet.
func HasPassword(ctx context.Context, db bun.IDB) (bool, error) {
	return db.NewSelect().Model((*domain.Setting)(nil)).Where("key = ?", domain.SettingPasswordHash).Exists(ctx)
}
