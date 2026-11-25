// jat-server is the only binary on the base: the API, the web app, the MCP over HTTP, and the admin's commands.
// The CLI, jat, is a client of its API.
package main

import (
	"fmt"
	"os"

	"github.com/cyril-fl/jat/config"
	cmdadmin "github.com/cyril-fl/jat/core/server/cmd-admin"
	cmdserve "github.com/cyril-fl/jat/core/server/cmd-serve"
	"github.com/cyril-fl/jat/core/services/process/createschema"
	"github.com/cyril-fl/jat/core/services/sqlite"
	plumcliutilsargs "github.com/cyril-fl/plum/cli/utils/args"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
)

var cmd = config.Current.Commands

// db is opened before every subcommand, with the schema made: it is idempotent.
var db *bun.DB

var cmdRoot = &cobra.Command{
	Use:           cmd.ServerRoot.Use,
	Short:         cmd.ServerRoot.Short,
	Version:       config.Version(),
	SilenceUsage:  true,
	SilenceErrors: true,

	PersistentPreRunE: func(cobra *cobra.Command, args []string) error {
		if !cobra.HasParent() || cobra.Name() == "completion" || cobra.Name() == "help" {
			return nil
		}
		path, _ := cobra.Root().PersistentFlags().GetString("db")
		var err error
		if db, err = sqlite.Open(path); err != nil {
			return err
		}
		schema := createschema.Process(cobra.Context(), db)
		schema.Run()
		return schema.GetErr()
	},
	PersistentPostRunE: func(cobra *cobra.Command, args []string) error {
		if db != nil {
			return db.Close()
		}
		return nil
	},
}

var cmdServe = &cobra.Command{
	Use:     cmd.Serve.Use,
	Short:   cmd.Serve.Short,
	Long:    cmd.Serve.Long,
	Example: cmd.Serve.Example,
	Args:    cobra.NoArgs,
	RunE: func(cobra *cobra.Command, args []string) error {
		if err := cmdadmin.EnsurePassword(cobra.Context(), db); err != nil {
			return err
		}
		return cmdserve.Serve(cobra, db)
	},
}

var cmdResetPassword = &cobra.Command{
	Use:     cmd.ResetPassword.Use,
	Short:   cmd.ResetPassword.Short,
	Long:    cmd.ResetPassword.Long,
	Example: cmd.ResetPassword.Example,
	Args:    cobra.NoArgs,
	RunE: func(cobra *cobra.Command, args []string) error {
		return cmdadmin.ResetPassword(cobra, db)
	},
}

var cmdBackup = &cobra.Command{
	Use:     cmd.Backup.Use,
	Short:   cmd.Backup.Short,
	Long:    cmd.Backup.Long,
	Example: cmd.Backup.Example,
	Args:    cobra.ExactArgs(1),
	RunE: func(cobra *cobra.Command, args []string) error {
		return cmdadmin.Backup(cobra, db, args[0])
	},
}

func init() {
	dbPath := os.Getenv(config.Env("DB"))
	if dbPath == "" {
		dbPath = "jat.db"
	}
	cmdRoot.PersistentFlags().String("db", dbPath, "SQLite file of the base, created when missing ("+config.Env("DB")+")")
	plumcliutilsargs.Makeflags(cmdServe, cmd.Serve.Flags)
	cmdRoot.AddCommand(cmdServe, cmdResetPassword, cmdBackup)
}

func main() {
	if err := cmdRoot.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
