// Package createuser is the sign-up of the multi mode: an account created by an invitation, or by anyone in the
// public mode. The first account of a base needs no invitation and runs the others (admin).
package createuser

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// MaxName is the most characters of an account's name.
const MaxName = 64

var (
	ErrNoName    = domain.Invalid("the account needs a name")
	ErrLongName  = domain.Invalid("the name of an account has at most 64 characters")
	ErrNameTaken = domain.Invalid("this name is taken")
	ErrBadInvite = domain.Forbidden("this invitation is unknown, used or expired")
)

// Input is a sign-up. Invite is the token of an invitation, needed in the invitation mode.
type Input struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Invite   string `json:"invite,omitempty" binding:"optional"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx    context.Context
	db     bun.IDB
	input  Input
	public bool
}

type cache struct {
	result domain.User
}

// Process creates the account of input; public tells the public mode (domain.ServerMode.Public), where no
// invitation is needed.
func Process(ctx context.Context, db bun.IDB, input Input, public bool) *run {
	return &run{
		ctx:    ctx,
		db:     db,
		input:  input,
		public: public,
	}
}

func (p *run) Run() {
	p.ensureInput()
	p.saveUser()
}

// Getters & Setters
func (p *run) GetResult() domain.User {
	return p.cache.result
}

// Methods
func (p *run) ensureInput() {
	p.input.Name = strings.TrimSpace(p.input.Name)
	switch {
	case p.input.Name == "":
		p.SetErr(ErrNoName)
	case utf8.RuneCountInString(p.input.Name) > MaxName:
		p.SetErr(ErrLongName)
	case utf8.RuneCountInString(p.input.Password) < setpassword.MinLength:
		p.SetErr(setpassword.ErrShortPassword)
	}
}

func (p *run) saveUser() {
	if p.HasErr() {
		return
	}
	now := time.Now().UTC()
	user := domain.User{ID: uuid.NewString(), Name: p.input.Name, PasswordHash: password.Hash(p.input.Password), CreatedAt: now}
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		count, err := tx.NewSelect().Model((*domain.User)(nil)).Count(ctx)
		if err != nil {
			return err
		}
		user.Admin = count == 0
		if !user.Admin && !p.public {
			// Used in the same statement that checks it: two sign-ups cannot share an invitation.
			result, err := tx.NewUpdate().Model((*domain.Invite)(nil)).Set("used_at = ?", now).
				Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", password.TokenHash(p.input.Invite), now).Exec(ctx)
			if err != nil {
				return err
			}
			if rows, _ := result.RowsAffected(); rows == 0 {
				return ErrBadInvite
			}
		}
		taken, err := tx.NewSelect().Model((*domain.User)(nil)).Where("lower(name) = lower(?)", user.Name).Exists(ctx)
		if err != nil {
			return err
		}
		if taken {
			return ErrNameTaken
		}
		_, err = tx.NewInsert().Model(&user).Exec(ctx)
		return err
	}))
	p.cache.result = user
}
