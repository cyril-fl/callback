package domain

import (
	"time"

	"github.com/uptrace/bun"
)

// TokenKind tells what a token is for.
type TokenKind string

const (
	// TokenSession is a login of the web app or the CLI: 30 days, started again at each use.
	TokenSession TokenKind = "session"
	// TokenAPI is made from the settings for a script or an agent: it never manages the tokens.
	TokenAPI TokenKind = "api"
	// TokenOAuth is the connection of an OAuth client (the MCP over HTTP): Hash is its access token, valid until
	// AccessExpiresAt, RefreshHash its refresh token, valid until ExpiresAt.
	TokenOAuth TokenKind = "oauth"
)

// Token lets a client in. The base keeps the SHA-256 of the token, never the token.
type Token struct {
	bun.BaseModel `bun:"table:tokens,alias:t" json:"-"`

	ID         string    `bun:",pk" json:"id"`
	UserID     string    `bun:",notnull,default:''" json:"-"`
	Kind       TokenKind `bun:",notnull" json:"kind" enums:"session,api,oauth"`
	Name       string    `bun:",notnull,default:''" json:"name"`
	Hash       string    `bun:",notnull,unique" json:"-"`
	ReadOnly   bool      `bun:",notnull,default:false" json:"read_only"`
	CreatedAt  time.Time `bun:",notnull" json:"created_at"`
	LastUsedAt time.Time `bun:",nullzero" json:"last_used_at,omitzero" binding:"optional"` // never used when absent
	ExpiresAt  time.Time `bun:",nullzero" json:"expires_at,omitzero" binding:"optional"`   // no limit when absent

	ClientID        string    `bun:",nullzero" json:"-"`
	RefreshHash     string    `bun:",nullzero" json:"-"`
	AccessExpiresAt time.Time `bun:",nullzero" json:"-"`
}

// OAuthCode is an authorization code between the consent and the token: it serves once, for a minute. The client
// proves with the verifier of CodeChallenge (PKCE S256) that it is the one which asked.
type OAuthCode struct {
	bun.BaseModel `bun:"table:oauth_codes,alias:oc"`

	CodeHash      string    `bun:",pk"`
	UserID        string    `bun:",notnull,default:''"`
	ClientID      string    `bun:",notnull"`
	RedirectURI   string    `bun:",notnull"`
	CodeChallenge string    `bun:",notnull"`
	ReadOnly      bool      `bun:",notnull"`
	ExpiresAt     time.Time `bun:",notnull"`
}

// LoginFailure counts the wrong passwords in a row of a key (an address): kept in the base, a restart forgets none.
type LoginFailure struct {
	bun.BaseModel `bun:"table:login_failures,alias:lf"`

	Key       string    `bun:",pk"`
	Count     int       `bun:",notnull"`
	LastAt    time.Time `bun:",notnull"`
	ExpiresAt time.Time `bun:",notnull"`
}
