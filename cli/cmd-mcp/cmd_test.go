package cmdmcp

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"testing"

	"github.com/cyril-fl/callback/cli/services/apiclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The ceiling of the tool list, paid by the model at every request: 6 tools of 4.1 KB at first, 4.8 KB with the
// interview and the contact, 7 of 5.2 KB with the status history. Complete tools that save round trips are worth bytes; optimise what exists before adding.
const maxTools, maxToolBytes = 8, 6000

var (
	readTools  = []string{"list_applications", "read_applications", "application_stats", "status_history"}
	writeTools = []string{"add_applications", "update_applications", "delete_applications"}
)

// announced lists the tools and the instructions as a client gets them at the connection; listing calls no route.
func announced(t *testing.T, readOnly bool) ([]*mcp.Tool, string) {
	t.Helper()
	ctx := context.Background()
	client, err := apiclient.NewClientWithResponses("http://unused")
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	session, err := NewServer(client, readOnly, "test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	agent, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer agent.Close()
	result, err := agent.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result.Tools, agent.InitializeResult().Instructions
}

// TestAddTools catches a schema the SDK cannot infer and wrong hints: the client runs a read-only tool freely and
// asks before a destructive one. It names every tool: one more is a choice. A read-only token lists the reads only.
func TestAddTools(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		tools, _ := announced(t, readOnly)
		var names []string
		for _, tool := range tools {
			names = append(names, tool.Name)
			if tool.InputSchema == nil || tool.Description == "" {
				t.Errorf("tool %q: schema %v, description %q", tool.Name, tool.InputSchema, tool.Description)
			}
			// MCP wants an object: a client on the Python SDK rejects the whole tool list otherwise.
			if schema, ok := tool.OutputSchema.(map[string]any); tool.OutputSchema != nil && (!ok || schema["type"] != "object") {
				t.Errorf("tool %q: output schema %v, want an object", tool.Name, tool.OutputSchema)
			}
			a := tool.Annotations
			destructive := a != nil && a.DestructiveHint != nil && *a.DestructiveHint
			switch {
			case a == nil:
				t.Errorf("tool %q: no annotations", tool.Name)
			case slices.Contains(readTools, tool.Name) != a.ReadOnlyHint:
				t.Errorf("tool %q: read-only hint %v", tool.Name, a.ReadOnlyHint)
			case !a.ReadOnlyHint && destructive != (tool.Name == "delete_applications"):
				t.Errorf("tool %q: destructive hint %v", tool.Name, destructive)
			}
		}
		want := readTools
		if !readOnly {
			want = append(slices.Clone(readTools), writeTools...)
		}
		if slices.Sort(names); !slices.Equal(names, slices.Sorted(slices.Values(want))) {
			t.Errorf("readOnly %v: tools = %v, want %v", readOnly, names, want)
		}
	}
}

// TestInstructions catches instructions naming a tool or an argument the server does not announce, renamed or only
// in the writing tools: the agent would look for it in vain.
func TestInstructions(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		tools, instructions := announced(t, readOnly)
		known := map[string]bool{"job_post": true, "source_detail": true, "interview_at": true} // values and columns, not arguments
		for _, tool := range tools {
			known[tool.Name] = true
			properties, _ := tool.InputSchema.(map[string]any)["properties"].(map[string]any)
			for name := range properties {
				known[name] = true
			}
		}
		for _, name := range regexp.MustCompile(`\b[a-z]+(?:_[a-z]+)+\b`).FindAllString(instructions, -1) {
			if !known[name] {
				t.Errorf("readOnly %v: the instructions name %q, which no tool announces", readOnly, name)
			}
		}
	}
}

// TestToolListSize measures what the tool list costs the model at every request: the name, description and input
// schema of each tool (clients do not give it the output schemas). go test -v -run Size prints it.
func TestToolListSize(t *testing.T) {
	tools, instructions := announced(t, false)
	size := 0
	for _, tool := range tools {
		data, err := json.Marshal(map[string]any{"name": tool.Name, "description": tool.Description, "input_schema": tool.InputSchema})
		if err != nil {
			t.Fatal(err)
		}
		size += len(data)
	}
	t.Logf("%d tools, %d bytes; instructions %d bytes", len(tools), size, len(instructions))
	if len(tools) > maxTools || size > maxToolBytes {
		t.Errorf("%d tools of %d bytes, over %d tools or %d bytes: optimise what exists before adding", len(tools), size, maxTools, maxToolBytes)
	}
}
