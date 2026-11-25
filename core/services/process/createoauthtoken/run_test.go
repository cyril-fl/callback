package createoauthtoken_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/createoauthcode"
	"github.com/cyril-fl/callback/core/services/process/createoauthtoken"
	"github.com/cyril-fl/callback/core/services/process/createsession"
	"github.com/cyril-fl/callback/core/services/process/deletetoken"
	"github.com/cyril-fl/callback/core/services/process/listtokens"
	"github.com/cyril-fl/callback/core/services/process/readtoken"
	"github.com/cyril-fl/callback/core/services/process/refreshoauthtoken"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	"github.com/cyril-fl/callback/core/services/testdb"
	"github.com/uptrace/bun"
)

// A connection from the consent to its revocation: a code that serves once and only with its verifier, an access token
// that reads as an oauth token, a refresh that kills the old tokens, a revocation that kills both.
func TestOAuthConnection(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	set := setpassword.Process(ctx, db, "correct horse")
	set.Run()
	if set.HasErr() {
		t.Fatal(set.GetErr())
	}

	verifier := strings.Repeat("v", 43)
	sum := sha256.Sum256([]byte(verifier))
	asked := createoauthcode.Input{ClientID: "client-a", RedirectURI: "https://chatgpt.com/cb", CodeChallenge: base64.RawURLEncoding.EncodeToString(sum[:]), ReadOnly: true}

	wrong := asked
	wrong.Password = "wrong"
	consent := createoauthcode.Process(ctx, db, wrong)
	consent.Run()
	if !errors.Is(consent.GetErr(), createsession.ErrBadCredentials) {
		t.Errorf("wrong password: err = %v, want %v", consent.GetErr(), createsession.ErrBadCredentials)
	}

	code := func() string {
		t.Helper()
		right := asked
		right.Password = "correct horse"
		consent := createoauthcode.Process(ctx, db, right)
		consent.Run()
		if consent.HasErr() {
			t.Fatalf("consent: %v", consent.GetErr())
		}
		return consent.GetResult()
	}
	exchange := createoauthtoken.Input{ClientID: "client-a", ClientName: "ChatGPT", RedirectURI: "https://chatgpt.com/cb", CodeVerifier: verifier}

	// Another client, another redirect, another verifier: refused, and the code is spent.
	for _, change := range []func(*createoauthtoken.Input){
		func(in *createoauthtoken.Input) { in.ClientID = "client-b" },
		func(in *createoauthtoken.Input) { in.RedirectURI = "https://evil.example/cb" },
		func(in *createoauthtoken.Input) { in.CodeVerifier = strings.Repeat("w", 43) },
	} {
		input := exchange
		input.Code = code()
		change(&input)
		process := createoauthtoken.Process(ctx, db, input)
		process.Run()
		if !errors.Is(process.GetErr(), createoauthtoken.ErrInvalidGrant) {
			t.Errorf("exchange %+v: err = %v, want %v", input, process.GetErr(), createoauthtoken.ErrInvalidGrant)
		}
	}
	if codes, _ := db.NewSelect().Model((*domain.OAuthCode)(nil)).Count(ctx); codes != 0 {
		t.Errorf("%d codes left after refused exchanges, want 0: a refused code is spent", codes)
	}

	// An expired code is refused.
	exchange.Code = code()
	if _, err := db.NewUpdate().Model((*domain.OAuthCode)(nil)).Set("expires_at = ?", time.Now().UTC().Add(-time.Second)).Where("1 = 1").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	expired := createoauthtoken.Process(ctx, db, exchange)
	expired.Run()
	if !errors.Is(expired.GetErr(), createoauthtoken.ErrInvalidGrant) {
		t.Errorf("expired code: err = %v, want %v", expired.GetErr(), createoauthtoken.ErrInvalidGrant)
	}

	exchange.Code = code()
	process := createoauthtoken.Process(ctx, db, exchange)
	process.Run()
	if process.HasErr() {
		t.Fatalf("exchange: %v", process.GetErr())
	}
	tokens := process.GetResult()
	if !strings.HasPrefix(tokens.AccessToken, "callback_") || tokens.RefreshToken == "" || tokens.ExpiresIn != 3600 || tokens.TokenType != "Bearer" {
		t.Errorf("tokens = %+v, want a callback_ access token of an hour and a refresh token", tokens)
	}
	again := createoauthtoken.Process(ctx, db, exchange)
	again.Run()
	if !errors.Is(again.GetErr(), createoauthtoken.ErrInvalidGrant) {
		t.Errorf("code used twice: err = %v, want %v", again.GetErr(), createoauthtoken.ErrInvalidGrant)
	}

	read := readtoken.Process(ctx, db, tokens.AccessToken)
	read.Run()
	if read.HasErr() || read.GetResult().Kind != domain.TokenOAuth || !read.GetResult().ReadOnly {
		t.Fatalf("access token: %+v, %v; want an oauth token, read-only", read.GetResult(), read.GetErr())
	}
	// The refresh token is no access token.
	if refused(ctx, db, tokens.RefreshToken) == nil {
		t.Errorf("refresh token used as an access token: read, want refused")
	}

	list := listtokens.Process(ctx, db)
	list.Run()
	if list.HasErr() || len(list.GetResult()) != 1 || list.GetResult()[0].Kind != domain.TokenOAuth || list.GetResult()[0].Name != "ChatGPT" || list.GetResult()[0].ExpiresAt.IsZero() {
		t.Fatalf("list = %+v, %v; want one OAuth connection named ChatGPT, with its expiry", list.GetResult(), list.GetErr())
	}

	// The access token dies after its hour, the connection stays.
	if _, err := db.NewUpdate().Model((*domain.Token)(nil)).Set("access_expires_at = ?", time.Now().UTC().Add(-time.Minute)).
		Where("kind = ?", domain.TokenOAuth).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if refused(ctx, db, tokens.AccessToken) == nil {
		t.Errorf("access token after its hour: read, want refused")
	}

	refresh := refreshoauthtoken.Process(ctx, db, refreshoauthtoken.Input{RefreshToken: tokens.RefreshToken, ClientID: "client-b"})
	refresh.Run()
	if !errors.Is(refresh.GetErr(), createoauthtoken.ErrInvalidGrant) {
		t.Errorf("refresh by another client: err = %v, want %v", refresh.GetErr(), createoauthtoken.ErrInvalidGrant)
	}
	refresh = refreshoauthtoken.Process(ctx, db, refreshoauthtoken.Input{RefreshToken: tokens.RefreshToken, ClientID: "client-a"})
	refresh.Run()
	if refresh.HasErr() {
		t.Fatalf("refresh: %v", refresh.GetErr())
	}
	fresh := refresh.GetResult()
	if fresh.AccessToken == tokens.AccessToken || fresh.RefreshToken == tokens.RefreshToken {
		t.Errorf("refresh gave the same tokens back")
	}
	if err := refused(ctx, db, fresh.AccessToken); err != nil {
		t.Errorf("new access token: %v", err)
	}
	old := refreshoauthtoken.Process(ctx, db, refreshoauthtoken.Input{RefreshToken: tokens.RefreshToken, ClientID: "client-a"})
	old.Run()
	if !errors.Is(old.GetErr(), createoauthtoken.ErrInvalidGrant) {
		t.Errorf("old refresh token: err = %v, want %v", old.GetErr(), createoauthtoken.ErrInvalidGrant)
	}

	revoke := deletetoken.Process(ctx, db, list.GetResult()[0].ID)
	revoke.Run()
	if revoke.HasErr() {
		t.Fatalf("revoke: %v", revoke.GetErr())
	}
	if refused(ctx, db, fresh.AccessToken) == nil {
		t.Errorf("access token after the revocation: read, want refused")
	}
	gone := refreshoauthtoken.Process(ctx, db, refreshoauthtoken.Input{RefreshToken: fresh.RefreshToken, ClientID: "client-a"})
	gone.Run()
	if !errors.Is(gone.GetErr(), createoauthtoken.ErrInvalidGrant) {
		t.Errorf("refresh after the revocation: err = %v, want %v", gone.GetErr(), createoauthtoken.ErrInvalidGrant)
	}
}

// PKCE S256 only, with a verifier of 43 to 128 characters (RFC 7636).
func TestVerifies(t *testing.T) {
	challenge := func(verifier string) string {
		sum := sha256.Sum256([]byte(verifier))
		return base64.RawURLEncoding.EncodeToString(sum[:])
	}
	good := strings.Repeat("a", 43)
	for _, c := range []struct {
		verifier, challenge string
		want                bool
	}{
		{good, challenge(good), true},
		{good, good, false}, // plain
		{strings.Repeat("a", 42), challenge(strings.Repeat("a", 42)), false},
		{strings.Repeat("a", 129), challenge(strings.Repeat("a", 129)), false},
		{good, "", false},
	} {
		if got := createoauthtoken.Verifies(c.verifier, c.challenge); got != c.want {
			t.Errorf("Verifies(%d chars, %q) = %v, want %v", len(c.verifier), c.challenge, got, c.want)
		}
	}
}

// refused is the error of reading an access token, nil when it works.
func refused(ctx context.Context, db bun.IDB, token string) error {
	read := readtoken.Process(ctx, db, token)
	read.Run()
	return read.GetErr()
}
