package cmdmcp

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/cyril-fl/callback/cli/services/apiclient"
	"github.com/cyril-fl/callback/cli/services/apiclient/apitest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// counter counts the requests the tools send to the API.
type counter struct{ requests *atomic.Int32 }

func (c counter) Do(r *http.Request) (*http.Response, error) {
	c.requests.Add(1)
	return http.DefaultClient.Do(r)
}

// connect serves the tools on the API of apitest to an MCP client in memory; the counter counts their requests.
func connect(t *testing.T, readOnly bool) (*mcp.ClientSession, *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	server := apitest.Serve(t)
	token := server.Session
	if readOnly {
		token = server.Token(t, true)
	}
	requests := &atomic.Int32{}
	client, err := apiclient.NewClientWithResponses(server.URL, apiclient.WithHTTPClient(counter{requests}), apiclient.Bearer(token))
	if err != nil {
		t.Fatal(err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	session, err := NewServer(client, readOnly, "test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	agent, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { agent.Close() })
	return agent, requests
}

// call runs a tool and decodes its structured answer into out, failing the test on an error.
func call(t *testing.T, agent *mcp.ClientSession, name string, args map[string]any, out any) {
	t.Helper()
	result, err := agent.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("%s: %v", name, result.Content[0].(*mcp.TextContent).Text)
	}
	if out != nil {
		data, _ := json.Marshal(result.StructuredContent)
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("%s: %v: %s", name, err, data)
		}
	}
}

func TestFactor(t *testing.T) {
	s := func(v string) *string { return &v }
	applied := apiclient.DomainStatus("applied")
	common, items := factor([]apiclient.DomainApplication{
		{Id: "a", Company: s("A"), Status: &applied, AppliedAt: s("2026-01-02")},
		{Id: "b", Company: s("B"), Status: &applied, AppliedAt: s("2026-01-02"), Comment: s("hi")},
	})
	if len(common) != 2 || common["status"] != "applied" || common["applied_at"] != "2026-01-02" {
		t.Errorf("common = %v", common)
	}
	if items[0]["status"] != nil || items[0]["company"] != "A" || items[1]["comment"] != "hi" || items[1]["id"] != "b" {
		t.Errorf("items = %v", items)
	}
	if common, items := factor(nil); common != nil || len(items) != 0 {
		t.Errorf("factor(nil) = %v, %v", common, items)
	}
}

// TestTools walks a batch through every tool, each call one request to the API whatever the batch's size.
func TestTools(t *testing.T) {
	agent, requests := connect(t, false)
	one := func(name string, args map[string]any, out any) {
		t.Helper()
		before := requests.Load()
		call(t, agent, name, args, out)
		if n := requests.Load() - before; n != 1 {
			t.Errorf("%s: %d requests to the API, want 1", name, n)
		}
	}

	var added apiclient.DomainBatchResult
	one("add_applications", map[string]any{"items": []map[string]any{
		{"company": "Acme", "location": "Paris", "applied_at": "2026-01-02"},
		{"company": "Globex", "location": "Lyon", "applied_at": "2026-01-03", "source": "recruiter", "source_detail": "hr@globex.test"},
		{"company": "Initech", "location": "Paris", "applied_at": "2026-01-04", "comment": "a long comment here"},
		{"company": "", "location": "Nowhere"},
	}}, &added)
	if added.Ids == nil || len(*added.Ids) != 3 || added.Failed == nil || len(*added.Failed) != 1 {
		t.Fatalf("add_applications = %+v %+v", added, *added.Failed)
	}
	ids := *added.Ids

	var listed listResult
	one("list_applications", map[string]any{"q": "paris"}, &listed)
	if listed.Total != 2 || len(listed.Applications) != 2 || listed.Applications[0].Location != nil || listed.Applications[0].Company == nil {
		t.Errorf("list_applications = %+v", listed)
	}
	var counted listResult
	one("list_applications", map[string]any{"count_only": true}, &counted)
	if counted.Total != 3 || len(counted.Applications) != 0 {
		t.Errorf("list_applications count_only = %+v", counted)
	}

	var read readResult
	one("read_applications", map[string]any{"ids": []string{ids[2], ids[0], "nope"}, "max_chars": 6}, &read)
	if len(read.Applications) != 2 || read.Applications[0]["id"] != ids[2] || read.Applications[0]["comment"] != "a long" ||
		read.Common["location"] != "Paris" || len(read.Missing) != 1 {
		t.Errorf("read_applications = %+v", read)
	}

	var updated apiclient.DomainBatchResult
	one("update_applications", map[string]any{"dry_run": true, "ops": []map[string]any{
		{"q": "paris", "set": map[string]any{"status": "rejected"}},
		{"ids": []string{ids[1]}, "set": map[string]any{"status": "interview", "rating": 4}},
	}}, &updated)
	if updated.Count != 3 || updated.DryRun == nil || !*updated.DryRun {
		t.Errorf("update_applications dry_run = %+v", updated)
	}
	updated = apiclient.DomainBatchResult{}
	one("update_applications", map[string]any{"summary": true, "ops": []map[string]any{
		{"q": "paris", "set": map[string]any{"status": "rejected"}},
		{"ids": []string{ids[1]}, "set": map[string]any{"status": "interview", "rating": 4}},
	}}, &updated)
	if updated.Count != 3 || updated.Ids != nil {
		t.Errorf("update_applications = %+v", updated)
	}

	var stats apiclient.ReadstatsStats
	one("application_stats", nil, &stats)
	if stats.Total != 3 || stats.ByStatus["rejected"] != 2 || stats.ByStatus["interview"] != 1 {
		t.Errorf("application_stats = %+v", stats)
	}

	var deleted apiclient.DomainBatchResult
	one("delete_applications", map[string]any{"status": []string{"rejected"}}, &deleted)
	if deleted.Count != 2 {
		t.Errorf("delete_applications = %+v", deleted)
	}
}
