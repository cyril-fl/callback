// Package cmdadmin is the admin's commands of callback-server: they run on the base, on the server's machine.
package cmdadmin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/setpassword"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"golang.org/x/term"
)

// DefaultUser is the name of the account EnsurePassword creates without CALLBACK_USER.
const DefaultUser = "admin"

// EnsurePassword creates the account of a base that has none, named CALLBACK_USER (DefaultUser when unset) with the
// password CALLBACK_PASSWORD: a server put up from a compose file is usable at once. On a base of one account,
// CALLBACK_USER renames it; otherwise it does nothing.
func EnsurePassword(ctx context.Context, db bun.IDB) error {
	var users []domain.User
	if err := db.NewSelect().Model(&users).Column("id", "name").Limit(2).Scan(ctx); err != nil {
		return err
	}
	name := strings.TrimSpace(os.Getenv(config.Env("USER")))
	switch {
	case len(users) == 1 && name != "" && name != users[0].Name:
		_, err := db.NewUpdate().Model((*domain.User)(nil)).Set("name = ?", name).Where("id = ?", users[0].ID).Exec(ctx)
		return err
	case len(users) > 0:
		return nil
	}
	pass := os.Getenv(config.Env("PASSWORD"))
	if pass == "" {
		return fmt.Errorf("the base has no password yet: set %s", config.Env("PASSWORD"))
	}
	if name == "" {
		name = DefaultUser
	}
	user := domain.User{ID: uuid.NewString(), Name: name, PasswordHash: "-", Admin: true, CreatedAt: time.Now().UTC()}
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&user).Exec(ctx); err != nil {
			return err
		}
		process := setpassword.Process(ctx, tx, user.ID, pass)
		process.Run()
		return process.GetErr()
	})
}

// ResetPassword changes the password of the account of --user, which a base of one account may leave out: from
// CALLBACK_PASSWORD, or asked on the terminal.
func ResetPassword(cobra *cobra.Command, db bun.IDB) error {
	name, _ := cobra.Flags().GetString("user")
	query := db.NewSelect().Model((*domain.User)(nil)).Column("id").Limit(2)
	if name != "" {
		query = query.Where("lower(name) = lower(?)", name)
	}
	var ids []string
	if err := query.Scan(cobra.Context(), &ids); err != nil {
		return err
	}
	switch {
	case len(ids) == 0 && name != "":
		return fmt.Errorf("no account named %s", name)
	case len(ids) == 0:
		return errors.New("the base has no account yet: serve creates it")
	case len(ids) > 1:
		return errors.New("the base has several accounts: name one with --user")
	}
	pass := os.Getenv(config.Env("PASSWORD"))
	if pass == "" {
		var err error
		if pass, err = ask(cobra); err != nil {
			return err
		}
	}
	process := setpassword.Process(cobra.Context(), db, ids[0], pass)
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
