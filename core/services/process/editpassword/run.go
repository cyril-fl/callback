// Package editpassword changes the password of the account logged in, given its current one, and logs its other
// devices out.
package editpassword

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

var ErrWrongPassword = domain.Invalid("the current password is wrong")

// Input is the password the account has, and the one it wants.
type Input struct {
	Current string `json:"current"`
	New     string `json:"new"`
}

type run struct {
	plumprimitive.Process
	ctx     context.Context
	db      bun.IDB
	input   Input
	session string
}

// Process gives the account of ctx a new password; its session session stays logged in.
func Process(ctx context.Context, db bun.IDB, input Input, session string) *run {
	return &run{
		ctx:     ctx,
		db:      db,
		input:   input,
		session: session,
	}
}

func (p *run) Run() {
	p.checkPassword()
	p.savePassword()
}

// Methods

func (p *run) checkPassword() {
	var user domain.User
	err := p.db.NewSelect().Model(&user).Column("password_hash").Where("id = ?", domain.UserID(p.ctx)).Scan(p.ctx)
	if err == nil && !password.Verify(p.input.Current, user.PasswordHash) {
		err = ErrWrongPassword
	}
	p.SetErr(err)
}

func (p *run) savePassword() {
	if p.HasErr() {
		return
	}
	process := setpassword.Process(p.ctx, p.db, domain.UserID(p.ctx), p.input.New).Keep(p.session)
	process.Run()
	p.SetErr(process.GetErr())
}
