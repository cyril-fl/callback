// Package renameuser changes the name of the account logged in, the one it logs in with.
package renameuser

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createuser"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// Input is the new name.
type Input struct {
	Name string `json:"name"`
}

type run struct {
	plumprimitive.Process
	ctx   context.Context
	db    bun.IDB
	input Input
}

// Process renames the account of ctx; the names stay unique, case aside.
func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.ensureInput()
	p.saveName()
}

// Methods

func (p *run) ensureInput() {
	p.input.Name = strings.TrimSpace(p.input.Name)
	switch {
	case p.input.Name == "":
		p.SetErr(createuser.ErrNoName)
	case utf8.RuneCountInString(p.input.Name) > createuser.MaxName:
		p.SetErr(createuser.ErrLongName)
	}
}

func (p *run) saveName() {
	if p.HasErr() {
		return
	}
	id := domain.UserID(p.ctx)
	p.SetErr(p.db.RunInTx(p.ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		taken, err := tx.NewSelect().Model((*domain.User)(nil)).Where("lower(name) = lower(?) AND id <> ?", p.input.Name, id).Exists(ctx)
		if err != nil {
			return err
		}
		if taken {
			return createuser.ErrNameTaken
		}
		_, err = tx.NewUpdate().Model((*domain.User)(nil)).Set("name = ?", p.input.Name).Where("id = ?", id).Exec(ctx)
		return err
	}))
}
