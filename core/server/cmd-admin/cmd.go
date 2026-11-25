// Package cmdadmin is the admin's commands of callback-server: they run on the base, on the server's machine.
package cmdadmin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"golang.org/x/term"
)

// EnsurePassword sets the password from CALLBACK_PASSWORD on a base that has none yet: a server put up from a compose
// file is usable at once. It does nothing once a password is set.
func EnsurePassword(ctx context.Context, db bun.IDB) error {
	has, err := setpassword.HasPassword(ctx, db)
	if err != nil || has {
		return err
	}
	pass := os.Getenv(config.Env("PASSWORD"))
	if pass == "" {
		return fmt.Errorf("the base has no password yet: set %s", config.Env("PASSWORD"))
	}
	process := setpassword.Process(ctx, db, pass)
	process.Run()
	return process.GetErr()
}

// ResetPassword changes the password: from CALLBACK_PASSWORD, or asked on the terminal.
func ResetPassword(cobra *cobra.Command, db bun.IDB) error {
	pass := os.Getenv(config.Env("PASSWORD"))
	if pass == "" {
		var err error
		if pass, err = ask(cobra); err != nil {
			return err
		}
	}
	process := setpassword.Process(cobra.Context(), db, pass)
	process.Run()
	if process.HasErr() {
		return process.GetErr()
	}
	fmt.Fprintln(cobra.OutOrStdout(), "Password changed: every session is logged out.")
	return nil
}

// Backup copies the base to file while the server runs: VACUUM INTO writes a clean copy in one read transaction.
// The image has no sqlite3 to do it.
func Backup(cobra *cobra.Command, db bun.IDB, file string) error {
	if _, err := os.Stat(file); !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s already exists", file)
	}
	if _, err := db.ExecContext(cobra.Context(), "VACUUM INTO ?", file); err != nil {
		return err
	}
	fmt.Fprintf(cobra.OutOrStdout(), "Base copied to %s\n", file)
	return nil
}

func ask(cobra *cobra.Command) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("set %s, or run on a terminal", config.Env("PASSWORD"))
	}
	fmt.Fprint(cobra.ErrOrStderr(), "New password: ")
	first, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cobra.ErrOrStderr())
	if err != nil {
		return "", err
	}
	fmt.Fprint(cobra.ErrOrStderr(), "Again: ")
	second, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cobra.ErrOrStderr())
	if err != nil {
		return "", err
	}
	if string(first) != string(second) {
		return "", errors.New("the passwords differ")
	}
	return string(first), nil
}
