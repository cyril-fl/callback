// Package createoauthtoken exchanges an OAuth authorization code for a new connection: a token of kind oauth.
package createoauthtoken

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/password"
	"github.com/cyril-fl/callback/core/services/process/createapitoken"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const (
	// AccessLifetime is how long an access token works: the client refreshes it after.
	AccessLifetime = time.Hour
	// RefreshLifetime is how long a connection lasts without use: each refresh starts it again.
	RefreshLifetime = 90 * 24 * time.Hour
)

// ErrInvalidGrant is the OAuth error invalid_grant: a code or refresh token unknown, expired, used, or of another client.
var ErrInvalidGrant = domain.Invalid("the code or refresh token is unknown, expired, already used, or of another client")

// Input is the exchange of an authorization code (grant_type=authorization_code).
type Input struct {
	Code         string
	ClientID     string
	ClientName   string // the name of the connection in the list of tokens
	RedirectURI  string
	CodeVerifier string
}

// Result is the token answer of OAuth: the tokens go to the client, the base keeps their hash.
type Result struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	input Input
}

type cache struct {
	code   domain.OAuthCode
	result Result
}

// Process exchanges an authorization code for a new connection: an access token of an hour and a refresh token of
// 90 days. The code serves once, used or refused.
func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.takeCode()
	p.ensureCode()
	p.saveToken()
}

// Getters & Setters
func (p *run) GetResult() Result {
	return p.cache.result
}

// Methods

// takeCode reads the code and deletes it: of two exchanges at once, only the one whose delete removed the row gets
// it. The row never changes, so reading it before the delete needs no transaction.
func (p *run) takeCode() {
	hash := password.TokenHash(p.input.Code)
	err := p.db.NewSelect().Model(&p.cache.code).Where("code_hash = ? AND expires_at > ?", hash, time.Now().UTC()).Scan(p.ctx)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrInvalidGrant
	}
	if err != nil {
		p.SetErr(err)
		return
	}
	deleted, err := p.db.NewDelete().Model((*domain.OAuthCode)(nil)).Where("code_hash = ?", hash).Exec(p.ctx)
	if err != nil {
		p.SetErr(err)
		return
	}
	if rows, _ := deleted.RowsAffected(); rows == 0 {
		p.SetErr(ErrInvalidGrant)
	}
}

func (p *run) ensureCode() {
	if p.HasErr() {
		return
	}
	code := p.cache.code
	if code.ClientID != p.input.ClientID || code.RedirectURI != p.input.RedirectURI || !Verifies(p.input.CodeVerifier, code.CodeChallenge) {
		p.SetErr(ErrInvalidGrant)
	}
}

func (p *run) saveToken() {
	if p.HasErr() {
		return
	}
	result, row := NewTokens()
	row.ID = uuid.NewString()
	row.Kind = domain.TokenOAuth
	row.Name = p.input.ClientName
	row.ReadOnly = p.cache.code.ReadOnly
	row.ClientID = p.cache.code.ClientID
	row.CreatedAt = time.Now().UTC()
	if _, err := p.db.NewInsert().Model(&row).Exec(p.ctx); err != nil {
		p.SetErr(err)
		return
	}
	p.cache.result = result
}

// Helpers

// NewTokens gives a new access token and refresh token, and the row fields they set: their hashes and expiries.
func NewTokens() (Result, domain.Token) {
	// UTC: SQLite compares the times as text.
	now := time.Now().UTC()
	result := Result{
		AccessToken:  createapitoken.Prefix + password.Token(),
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessLifetime.Seconds()),
		RefreshToken: password.Token(),
	}
	row := domain.Token{
		Hash:            password.TokenHash(result.AccessToken),
		RefreshHash:     password.TokenHash(result.RefreshToken),
		AccessExpiresAt: now.Add(AccessLifetime),
		ExpiresAt:       now.Add(RefreshLifetime),
	}
	return result, row
}

// Verifies tells if verifier is the one of challenge, PKCE S256: BASE64URL(SHA256(verifier)).
func Verifies(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}
