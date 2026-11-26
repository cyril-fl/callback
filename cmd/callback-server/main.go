// callback-server is the only binary on the base: the API, the web app, the MCP over HTTP, and the admin's commands.
// The CLI, callback, is a client of its API.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/domain"
	cmdadmin "github.com/cyril-fl/callback/core/server/cmd-admin"
	cmdserve "github.com/cyril-fl/callback/core/server/cmd-serve"
	"github.com/cyril-fl/callback/core/server/mcphttp"
	"github.com/cyril-fl/callback/core/server/oauth"
	"github.com/cyril-fl/callback/core/services/database"
	"github.com/cyril-fl/callback/core/services/process/createschema"
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
		if db, err = database.Open(path); err != nil {
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
	// The MCP over HTTP and its OAuth share the --mcp-addr listener; the tools call the API in process.
	cmdserve.MCP = func(mux *http.ServeMux, db bun.IDB, api http.Handler, version string, mode domain.ServerMode) {
		mux.Handle("/mcp", mcphttp.Handler(api, version))
		oauth.Routes(mux, db, mode.Multi())
	}

	dbPath := os.Getenv(config.Env("DB"))
	if dbPath == "" {
		dbPath = "callback.db"
	}
	cmdRoot.PersistentFlags().String("db", dbPath, "SQLite file of the base, created when missing, or a Postgres URL: postgres://user:password@host:5432/callback?sslmode=disable ("+config.Env("DB")+")")
	plumcliutilsargs.Makeflags(cmdServe, cmd.Serve.Flags)
	plumcliutilsargs.Makeflags(cmdResetPassword, cmd.ResetPassword.Flags)
	cmdRoot.AddCommand(cmdServe, cmdResetPassword, cmdBackup)
}

func main() {
	if err := cmdRoot.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
