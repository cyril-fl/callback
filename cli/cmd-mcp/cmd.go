// Package cmdmcp serves the applications to an agent over MCP, a client of the API like the rest of the CLI: the
// rules (statuses, dates, batches) are those of the server.
package cmdmcp

import (
	"context"

	"github.com/cyril-fl/callback/cli/services/apiclient"
	"github.com/cyril-fl/callback/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// Serve speaks MCP on stdin and stdout: nothing else may be printed there.
func Serve(cobra *cobra.Command, _ []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	// A read-only token in CALLBACK_SESSION gets the reading tools only, as over HTTP.
	me, err := apiclient.Decode[apiclient.LoginMeResult](client.GetApiMe(cobra.Context()))
	if err != nil {
		return err
	}
	return NewServer(client, me.ReadOnly, cobra.Root().Version).Run(cobra.Context(), &mcp.StdioTransport{})
}

// NewServer is the MCP server of the tools acting through client, the reading ones only when readOnly.
func NewServer(client *apiclient.ClientWithResponses, readOnly bool, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: config.Current.Name, Version: version}, &mcp.ServerOptions{Instructions: Instructions(readOnly)})
	AddTools(server, client, readOnly)
	return server
}

// AddTools gives the server the tools acting through client; the reading ones only when readOnly. Each tool call is
// one request to the API, so a batch is one transaction.
func AddTools(server *mcp.Server, client *apiclient.ClientWithResponses, readOnly bool) {
	addTool(server, readOnly, &mcp.Tool{
		Name:        "list_applications",
		Annotations: reads(),
		Description: "List the applications matching the filters, compact (id, company, status, applied_at), with the total.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listArgs) (*mcp.CallToolResult, listResult, error) {
		out, err := list(ctx, client, args)
		return nil, out, err
	})

	addTool(server, readOnly, &mcp.Tool{
		Name:        "read_applications",
		Annotations: reads(),
		Description: "Read applications whole by id, in the order asked, what they share said once.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args readArgs) (*mcp.CallToolResult, readResult, error) {
		out, err := read(ctx, client, args)
		return nil, out, err
	})

	addTool(server, readOnly, &mcp.Tool{
		Name:        "application_stats",
		Annotations: reads(),
		Description: "Count the applications per status and per source (reuse these names), with the response rate.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, apiclient.ReadstatsStats, error) {
		stats, err := apiclient.Decode[apiclient.ReadstatsStats](client.GetApiStats(ctx))
		return nil, stats, err
	})

	addTool(server, readOnly, &mcp.Tool{
		Name:        "status_history",
		Annotations: reads(),
		Description: "The statuses applications reached and when, oldest first: delays, what moved lately.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args historyArgs) (*mcp.CallToolResult, historyResult, error) {
		params := &apiclient.GetApiStatusChangesParams{Since: args.Since}
		if len(args.IDs) > 0 {
			params.Ids = &args.IDs
		}
		changes, err := apiclient.Decode[[]apiclient.DomainStatusChange](client.GetApiStatusChanges(ctx, params))
		return nil, historyResult{Changes: changes}, err
	})
	addTool(server, readOnly, &mcp.Tool{
		Name:        "add_applications",
		Annotations: writes(false, false),
		Description: "Create applications in one batch; answers their ids and the items that failed.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args addArgs) (*mcp.CallToolResult, apiclient.DomainBatchResult, error) {
		result, err := apiclient.Decode[apiclient.DomainBatchResult](client.PostApiApplications(ctx, args.Items))
		return nil, result, err
	})

	addTool(server, readOnly, &mcp.Tool{
		Name:        "update_applications",
		Annotations: writes(false, false),
		Description: "Change applications in one batch, each op picking them by ids or filters and giving the fields to set.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args apiclient.UpdateapplicationsInput) (*mcp.CallToolResult, apiclient.DomainBatchResult, error) {
		result, err := apiclient.Decode[apiclient.DomainBatchResult](client.PatchApiApplications(ctx, args))
		return nil, result, err
	})

	addTool(server, readOnly, &mcp.Tool{
		Name:        "delete_applications",
		Annotations: writes(true, true),
		Description: "Delete for good the applications of ids or filters.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args apiclient.DeleteapplicationsInput) (*mcp.CallToolResult, apiclient.DomainBatchResult, error) {
		result, err := apiclient.Decode[apiclient.DomainBatchResult](client.DeleteApiApplications(ctx, args))
		return nil, result, err
	})
}
