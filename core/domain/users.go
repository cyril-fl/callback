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
	Preferences *Preferences `bun:"type:json,nullzero,notnull,default:'{}'" json:"preferences,omitempty" binding:"optional"`
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

// Modes are how many accounts a server holds, the first one by default: one, whose login is the password alone, or
// several, each logging in with its name, created by sign-up (AddUserModes).
var Modes = []string{"single", "multi"}

// AddUserModes are the ways an account is created in the multi mode, the first one by default: by invitation of an
// admin only, or by anyone reaching the web app.
var AddUserModes = []string{"invitation", "public"}

// ServerMode is how the server runs, which the login and sign-up pages read: one of Modes, one of AddUserModes.
type ServerMode struct {
	Mode        string `json:"mode" enums:"single,multi"`
	AddUserMode string `json:"add_user_mode" enums:"invitation,public"`
}

// Multi tells whether the server holds several accounts.
func (m ServerMode) Multi() bool {
	return m.Mode == Modes[1]
}

// Public tells whether anyone may create an account.
func (m ServerMode) Public() bool {
	return m.AddUserMode == AddUserModes[1]
}

// Invite lets someone create an account in the invitation mode, once, for InviteLifetime: the base keeps the hash of
// its token, which goes to the admin who made it only.
type Invite struct {
	bun.BaseModel `bun:"table:invites,alias:iv" json:"-"`

	ID        string    `bun:",pk" json:"id"`
	TokenHash string    `bun:",notnull,unique" json:"-"`
	UserID    string    `bun:",notnull" json:"created_by"` // the admin who made it
	CreatedAt time.Time `bun:",notnull" json:"created_at"`
	ExpiresAt time.Time `bun:",notnull" json:"expires_at"`
	UsedAt    time.Time `bun:",nullzero" json:"-"` // set when it created an account
}

// InviteLifetime is how long an invitation waits for its sign-up.
const InviteLifetime = 7 * 24 * time.Hour
