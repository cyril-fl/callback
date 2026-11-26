package cmdmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/cyril-fl/callback/cli/services/apiclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// pageSize is how many applications a list gives when the agent names no limit: years of applications are more
// than an agent should read at once.
const pageSize = 50

// maxRead is the most ids read_applications takes: the comments make a big answer.
const maxRead = 50

// compact is the columns of a list when the agent asks no more; the API always adds id.
var compact = []string{"company", "title", "status", "applied_at"}

// addTool adds tool to server, unless readOnly and the tool changes something: its annotations tell it.
func addTool[In, Out any](server *mcp.Server, readOnly bool, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	if readOnly && !tool.Annotations.ReadOnlyHint {
		return
	}
	mcp.AddTool(server, tool, handler)
}

// reads marks a tool that changes nothing, so a client runs it without asking.
func reads() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)}
}

// writes marks a tool that changes the applications: destructive when it deletes (the client asks first),
// idempotent when calling it again with the same arguments changes nothing more.
func writes(destructive, idempotent bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: new(false)}
}

type listArgs struct {
	Status         []apiclient.GetApiApplicationsParamsStatus `json:"status,omitempty" jsonschema:"any of these"`
	Source         *string                                    `json:"source,omitempty" jsonschema:"spontaneous, job_post, recruiter or any name"`
	Location       *string                                    `json:"location,omitempty" jsonschema:"this city"`
	Q              *string                                    `json:"q,omitempty" jsonschema:"text in company, title, address, website, source_detail, contact or comment"`
	AppliedBefore  *string                                    `json:"applied_before,omitempty"`
	AppliedAfter   *string                                    `json:"applied_after,omitempty"`
	UntouchedSince *string                                    `json:"untouched_since,omitempty" jsonschema:"no status change nor follow-up since this day"`
	Fields         []string                                   `json:"fields,omitempty" jsonschema:"columns added: location, street, postal_code, country, website, source, source_detail, rating, comment, interview_at, contact_name, contact_email, created_at, updated_at, followed_up_at"`
	CountOnly      bool                                       `json:"count_only,omitempty" jsonschema:"the total only"`
	Sort           *string                                    `json:"sort,omitempty" jsonschema:"applied_at, company, location, source, status, rating, followed_up_at, created_at or updated_at; - first to reverse"`
	Limit          *int                                       `json:"limit,omitempty" jsonschema:"50 by default, 0 for all"`
	Offset         int                                        `json:"offset,omitempty"`
}

type statsArgs struct {
	Source        *string `json:"source,omitempty"`
	AppliedBefore *string `json:"applied_before,omitempty"`
	AppliedAfter  *string `json:"applied_after,omitempty"`
}

type listResult struct {
	Total        int                           `json:"total"`
	Applications []apiclient.DomainApplication `json:"applications,omitempty"`
}

// list is one GET: the filters, the columns and the page are all applied by the server.
func list(ctx context.Context, client *apiclient.ClientWithResponses, args listArgs) (listResult, error) {
	limit := pageSize
	if args.Limit != nil {
		limit = max(*args.Limit, 0)
	}
	params := &apiclient.GetApiApplicationsParams{
		Source: args.Source, Location: args.Location, Q: args.Q, AppliedBefore: args.AppliedBefore, AppliedAfter: args.AppliedAfter, UntouchedSince: args.UntouchedSince,
		Sort: args.Sort, Limit: &limit, Offset: &args.Offset, CountOnly: &args.CountOnly,
	}
	if len(args.Status) > 0 {
		params.Status = &args.Status
	}
	if !args.CountOnly {
		fields := slices.Compact(slices.Sorted(slices.Values(append(slices.Clone(compact), args.Fields...))))
		params.Fields = &fields
	}
	applications, total, err := apiclient.Paged[[]apiclient.DomainApplication](client.GetApiApplications(ctx, params))
	return listResult{Total: total, Applications: applications}, err
}

type readArgs struct {
	IDs      []string `json:"ids" jsonschema:"50 at most"`
	MaxChars int      `json:"max_chars,omitempty" jsonschema:"cut each comment to this many characters; truncated says so"`
}

// readResult is the applications asked, the fields they all share moved to common; missing is the ids not found.
type readResult struct {
	Common       map[string]any   `json:"common,omitempty"`
	Applications []map[string]any `json:"applications"`
	Missing      []string         `json:"missing,omitempty"`
}

// read is one GET for the whole batch, the comments cut on the server.
func read(ctx context.Context, client *apiclient.ClientWithResponses, args readArgs) (readResult, error) {
	if len(args.IDs) == 0 || len(args.IDs) > maxRead {
		return readResult{}, fmt.Errorf("give 1 to %d ids", maxRead)
	}
	all := 0
	params := &apiclient.GetApiApplicationsParams{Ids: &args.IDs, Limit: &all}
	if args.MaxChars > 0 {
		params.MaxChars = &args.MaxChars
	}
	found, err := apiclient.Decode[[]apiclient.DomainApplication](client.GetApiApplications(ctx, params))
	if err != nil {
		return readResult{}, err
	}
	byID := map[string]apiclient.DomainApplication{}
	for _, application := range found {
		byID[application.Id] = application
	}
	var ordered []apiclient.DomainApplication
	var missing []string
	for _, id := range args.IDs {
		if application, ok := byID[id]; ok {
			ordered = append(ordered, application)
		} else {
			missing = append(missing, id)
		}
	}
	common, applications := factor(ordered)
	return readResult{Common: common, Applications: applications, Missing: missing}, nil
}

// factor moves a field every application of several gives alike (the same status, source, day…) to common, said
// once instead of on each.
func factor(applications []apiclient.DomainApplication) (map[string]any, []map[string]any) {
	items := make([]map[string]any, 0, len(applications))
	for _, application := range applications {
		data, _ := json.Marshal(application)
		var item map[string]any
		_ = json.Unmarshal(data, &item)
		items = append(items, item)
	}
	if len(items) < 2 {
		return nil, items
	}
	common := map[string]any{}
	for key, value := range items[0] {
		// Every value is a JSON scalar, comparable with ==.
		if key != "id" && !slices.ContainsFunc(items[1:], func(item map[string]any) bool { other, ok := item[key]; return !ok || other != value }) {
			common[key] = value
		}
	}
	for _, item := range items {
		for key := range common {
			delete(item, key)
		}
	}
	return common, items
}

type historyArgs struct {
	IDs   []string `json:"ids,omitempty" jsonschema:"of these applications"`
	Since *string  `json:"since,omitempty" jsonschema:"reached on this day or after"`
}

// historyResult wraps the changes: a tool answers an object.
type historyResult struct {
	Changes []apiclient.DomainStatusChange `json:"changes"`
}

type addArgs struct {
	Items []apiclient.DomainNewApplication `json:"items" jsonschema:"500 at most"`
}
