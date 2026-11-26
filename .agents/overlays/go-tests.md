callback (one Go module at the root, SQLite).

- Run: `make test` (`go vet ./... && go test ./...`). Narrow with `go test ./core/services/process/<name>/ -run TestX`: no Makefile environment to keep.
- A process test takes its own in-memory base: `db := testdb.New(t)` (`core/services/testdb`), schema included. Pass `db` or a `bun.Tx` from it, as the process takes a `bun.IDB`.
- A route test runs the real API: `testserver.Serve(t)` (`core/services/testserver`), logged in; `server.Token(t, readOnly)` for an API token.
- A CLI or MCP test: `apitest.Serve(t)` (`cli/services/apiclient/apitest`) points the CLI at the test server; `apitest.Stdout(t, func() error {...})` captures the output.
- Never query the `*bun.DB` while a transaction from it is open (one connection: it hangs). Times in UTC.
- `cli/cmd-mcp/cmd_test.go` holds the MCP budget (tool count and bytes): a failure there is a design question, raise the budget rather than trim schemas.
- Web unit tests: `cd web && bun run test` (`test/*.test.ts`, on `app/utils/`).
