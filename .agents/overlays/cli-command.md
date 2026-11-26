callback.

- Binaries: `callback` (`cmd/callback/`, the CLI and `callback mcp`) is a client of the API only: `go list -deps ./cmd/callback` must never show bun nor `core/` (CI checks it). `callback-server` (`cmd/callback-server/`, `core/server/`) is the only one on the base.
- Commands, help and flags: `config/config.yaml` (built in), read through `config/config.go`.
- Commands: `cli/cmd-applications/cmd.go` (add, ls, show, edit, rm, stats: each one a single request, so a batch is one transaction), `cli/cmd-account/`. Output through `cli/services/output` (`Table`, `JSON`).
- Client: `cli/services/apiclient` (`New`, `Do`, `Decode`, `Paged`), generated part `client.gen.go` from `make openapi`.
- Writes are batches on the same routes as the web app: `POST /api/applications` takes an array, `PATCH` takes `ops`, `DELETE` ids or filters.
- MCP: `cli/cmd-mcp/`, few tools, all batch; `cli/cmd-mcp/cmd_test.go` holds the budget (tool count and bytes): optimise what exists first, raise the budget rather than trim a complete schema. `MCP.md` (in French) documents the tools.
- Tests: `apitest.Serve(t)` then `apitest.Stdout(t, ...)` (see the `go-tests` overlay).
