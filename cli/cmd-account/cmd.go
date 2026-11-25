// Package cmdaccount is the session and the tokens: callback login, logout, whoami and token.
package cmdaccount

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/cyril-fl/callback/cli/services/apiclient"
	"github.com/cyril-fl/callback/cli/services/output"
	"github.com/cyril-fl/callback/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Login opens a session on the server of args[0] with its password, and keeps it in the session file.
func Login(cobra *cobra.Command, args []string) error {
	password, err := readPassword()
	if err != nil {
		return err
	}
	server := strings.TrimRight(args[0], "/")
	client, err := apiclient.New(server)
	if err != nil {
		return err
	}
	result, err := apiclient.Decode[apiclient.CreatesessionResult](client.PostApiLogin(cobra.Context(), apiclient.CreatesessionInput{Password: password, Token: new(true)}))
	if err != nil {
		return err
	}
	if result.Token == nil {
		return errors.New("the server gave no token")
	}
	if err := (apiclient.Session{URL: server, Token: *result.Token}).Save(); err != nil {
		return err
	}
	fmt.Println("Logged in to", server)
	return nil
}

// readPassword takes CALLBACK_PASSWORD, else asks it on the terminal without echo.
func readPassword() (string, error) {
	if password := os.Getenv(config.Env("PASSWORD")); password != "" {
		return password, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("set %s, or run on a terminal", config.Env("PASSWORD"))
	}
	fmt.Fprint(os.Stderr, "Password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(password), err
}

// Logout ends the session on the server and removes the session file; a CALLBACK_SESSION token is left alone on disk, it
// was never there.
func Logout(cobra *cobra.Command, _ []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	response, err := client.PostApiLogoutWithResponse(cobra.Context())
	if err != nil {
		return err
	}
	// 401: the session had already expired, the file still goes.
	if code := response.StatusCode(); code != http.StatusNoContent && code != http.StatusUnauthorized {
		return apiclient.Fail(response.HTTPResponse, response.Body)
	}
	if strings.TrimSpace(os.Getenv(config.Env("SESSION"))) == "" {
		file, err := apiclient.SessionFile()
		if err != nil {
			return err
		}
		if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	fmt.Println("Logged out")
	return nil
}

// Whoami prints the token in use (GET /api/me) and its server.
func Whoami(cobra *cobra.Command, _ []string) error {
	session := apiclient.Read()
	client, err := session.Client()
	if err != nil {
		return err
	}
	me, err := apiclient.Decode[apiclient.LoginMeResult](client.GetApiMe(cobra.Context()))
	if err != nil {
		return err
	}
	if asJSON, _ := cobra.Flags().GetBool("json"); asJSON {
		return output.JSON(struct {
			apiclient.LoginMeResult
			Server string `json:"server"`
		}{me, session.URL})
	}
	who := string(me.Kind)
	if me.Name != nil {
		who += " " + *me.Name
	}
	if me.ReadOnly {
		who += " (read-only)"
	}
	fmt.Printf("%s on %s, server %s\n", who, session.URL, me.Version)
	return nil
}

// TokenAdd creates an API token and prints it, alone on stdout for TOKEN=$(callback token add name).
func TokenAdd(cobra *cobra.Command, args []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	readOnly, _ := cobra.Flags().GetBool("read-only")
	expires, _ := cobra.Flags().GetString("expires")
	in := apiclient.CreateapitokenInput{Name: args[0], ReadOnly: &readOnly, ExpiresIn: (*apiclient.CreateapitokenInputExpiresIn)(&expires)}
	result, err := apiclient.Decode[apiclient.CreateapitokenResult](client.PostApiTokens(cobra.Context(), in))
	if err != nil {
		return err
	}
	if asJSON, _ := cobra.Flags().GetBool("json"); asJSON {
		return output.JSON(result)
	}
	fmt.Fprintln(os.Stderr, "Keep this token: it is shown once.")
	fmt.Println(result.Token)
	return nil
}

// TokenList lists the sessions and tokens.
func TokenList(cobra *cobra.Command, _ []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	tokens, err := apiclient.Decode[[]apiclient.DomainToken](client.GetApiTokens(cobra.Context()))
	if err != nil {
		return err
	}
	if asJSON, _ := cobra.Flags().GetBool("json"); asJSON {
		return output.JSON(tokens)
	}
	rows := make([][]string, 0, len(tokens))
	for _, token := range tokens {
		rows = append(rows, []string{token.Id, token.Name, string(token.Kind), strconv.FormatBool(token.ReadOnly), or(token.ExpiresAt, "never"), or(token.LastUsedAt, "never")})
	}
	return output.Table([]string{"id", "name", "kind", "read_only", "expires_at", "last_used_at"}, rows)
}

// TokenRemove revokes a token.
func TokenRemove(cobra *cobra.Command, args []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	if _, err := apiclient.Decode[struct{}](client.DeleteApiTokensId(cobra.Context(), args[0])); err != nil {
		return err
	}
	fmt.Println("Revoked", args[0])
	return nil
}

func or(value *string, otherwise string) string {
	if value == nil {
		return otherwise
	}
	return *value
}
