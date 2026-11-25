// jat is the CLI of the job application tracker, a client of the API of jat-server: it never opens the base.
package main

import (
	"fmt"
	"os"

	cmdaccount "github.com/cyril-fl/jat/cli/cmd-account"
	cmdapplications "github.com/cyril-fl/jat/cli/cmd-applications"
	cmdmcp "github.com/cyril-fl/jat/cli/cmd-mcp"
	"github.com/cyril-fl/jat/config"
	plumcliconfig "github.com/cyril-fl/plum/cli/config"
	plumcliutilsargs "github.com/cyril-fl/plum/cli/utils/args"
	"github.com/spf13/cobra"
)

var cmd = config.Current.Commands

// command makes the command of params run by run, with its flags of the config.
func command(params plumcliconfig.Cmdparams, args cobra.PositionalArgs, run func(*cobra.Command, []string) error) *cobra.Command {
	c := &cobra.Command{Use: params.Use, Short: params.Short, Long: params.Long, Example: params.Example, Args: args, RunE: run}
	plumcliutilsargs.Makeflags(c, params.Flags)
	return c
}

// newRoot builds the command tree, fresh for each run of the tests: cobra keeps the flags set.
func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           cmd.Root.Use,
		Short:         cmd.Root.Short,
		Long:          cmd.Root.Long,
		Version:       config.Version(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	token := &cobra.Command{Use: cmd.Token.Use, Short: cmd.Token.Short}
	token.AddCommand(
		command(cmd.TokenAdd, cobra.ExactArgs(1), cmdaccount.TokenAdd),
		command(cmd.TokenLs, cobra.NoArgs, cmdaccount.TokenList),
		command(cmd.TokenRm, cobra.ExactArgs(1), cmdaccount.TokenRemove),
	)
	root.AddCommand(
		command(cmd.Login, cobra.ExactArgs(1), cmdaccount.Login),
		command(cmd.Logout, cobra.NoArgs, cmdaccount.Logout),
		command(cmd.Whoami, cobra.NoArgs, cmdaccount.Whoami),
		command(cmd.Add, cobra.NoArgs, cmdapplications.Add),
		command(cmd.Ls, cobra.NoArgs, cmdapplications.List),
		command(cmd.Show, cobra.MinimumNArgs(1), cmdapplications.Show),
		command(cmd.Edit, cobra.ArbitraryArgs, cmdapplications.Edit),
		command(cmd.Rm, cobra.ArbitraryArgs, cmdapplications.Remove),
		command(cmd.Stats, cobra.NoArgs, cmdapplications.Stats),
		command(cmd.MCP, cobra.NoArgs, cmdmcp.Serve),
		token,
	)
	return root
}

func main() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
