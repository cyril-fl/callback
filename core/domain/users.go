package domain

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// User is an account: every application, token and OAuth code belongs to one.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u" json:"-"`

	ID           string `bun:",pk" json:"id"`
	Name         string `bun:",notnull" json:"name"` // unique case-insensitive, see createschema
	PasswordHash string `bun:",notnull" json:"-"`
	Admin        bool   `bun:",notnull,default:false" json:"admin"`
	// Preferences are stored as JSON: readpreferences reads them over DefaultPreferences, so a preference added since
	// keeps its default. A pointer, so an insert without them writes the default {}, not zeros.
	Preferences *Preferences `bun:"type:json,nullzero,notnull,default:'{}'" json:"preferences,omitempty"`
	CreatedAt   time.Time    `bun:",notnull" json:"created_at"`
}

type userKey struct{}

// WithUser gives ctx the account the processes act for: the API from the request's token, serve's jobs from each account.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userKey{}, userID)
}

// UserID is the account of ctx, "" without one. A bun.Tx hands its RunInTx functions the ctx it began with, not the
// one given: begin transactions with a ctx that has the account.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(userKey{}).(string)
	return id
}
