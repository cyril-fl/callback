# callback

A job application tracker: one Go module at the root (`github.com/cyril-fl/callback`: the API, the CLI and the MCP) and a Nuxt web app (`web/`), on SQLite. Single account. The patterns come from tide (`~/Devloppement/go/notes/main`), without its accounts, Postgres and S3.

- `cmd/`: the two binaries. `callback-server` (`cmd/callback-server/`, `core/server/`) is the only one on the base: `serve` (the API, the web app with `--web`, the MCP over HTTP and its OAuth with `--mcp-addr`), `reset-password`, `backup <file>`. `callback` (`cmd/callback/`, `cli/`) is the CLI and the stdio MCP (`callback mcp`), a client of the API only: `go list -deps ./cmd/callback` never shows bun nor `core/` (CI checks it).
- `config/`: `config.yaml`, the commands, their help and flags, built in.
- `core/domain/`: the models; their json tags are the API's JSON and the web app's types. `core/services/process/`: the business logic, one plum Process each. `core/api/`: one folder per route, wired in `router.go`. `core/services/appfilter`: the WHERE of a `domain.Filter`, shared by the list, the batch update and the batch delete.
- `docs/swagger.yaml`: the OpenAPI, generated (`make openapi`, which also generates `cli/services/apiclient/client.gen.go`; then `bun run types` in `web/`). `MCP.md`: the MCP for agents (in French).

## Run

- `CALLBACK_PASSWORD=change-me go run ./cmd/callback-server serve --db ./data/callback.db --origin http://localhost:3000 --mcp-addr :8081`, then `cd web && bun run dev`. `CALLBACK_PASSWORD` sets the password of an empty base only; `callback-server reset-password` changes it.
- `make test`: `go vet` and `go test`, every test on an in-memory SQLite of its own (`core/services/testdb`, `core/services/testserver` for the API).
- `callback-server backup <file>` before touching data or the schema.

## Go rules

- Processes follow the `plum-process` skill of tide: guarded steps, `HasErr()` then `GetErr()`, `bun.IDB` so tests pass a base or a transaction.
- The base has one connection (`core/services/sqlite`): never query the `*bun.DB` while a transaction from it is open, use the `bun.Tx`. Times are stored in UTC (`time.Now().UTC()`): SQLite compares them as text.
- **No migration tool.** `createschema` creates missing tables and indexes and the columns a model gained, at every start. A rename, a type change or data to rewrite goes into `core/services/process/createschema/migrations.go` (append only). A new NOT NULL column needs a default. The columns of `applications` have no SQL default: bun leaves a defaulted column out of a batch insert when the first row has its zero value.
- Errors: a process's sentinel errors are made with `domain.Invalid/Unauthorized/Forbidden/NotFound`, which `respond` maps to 400/401/403/404; anything else is a 500 logged without detail to the client.
- Writes are batches: `POST /api/applications` takes an array, `PATCH` takes `ops`, `DELETE` ids or filters. One transaction, valid items written, invalid ones in `failed`; an empty filter is refused (it would match everything). The web app, the CLI and the MCP use these same routes.
- Auth (`core/api/services/auth`): a session (cookie `callback_session` or Bearer), an API token (`callback_…`) or an OAuth connection. Tokens other than sessions cannot manage tokens, log out, purge (`DELETE /api/data`) nor delete from the trash for good (`DELETE /api/applications/deleted`); a read-only token only GETs and its answers carry `Callback-Read-Only: true`. Wrong passwords make the address wait (`throttle`, shared with the OAuth consent page).
- The MCP has few tools, all batch: one tool call is one API request. `cli/cmd-mcp/cmd_test.go` holds the budget (tool count and bytes): optimise what exists before adding.

## Working with the user

- Talk to the user in French. Code, comments, commit messages in English; README in English, `MCP.md` in French.
- One conventional commit per change (`feat:`/`fix:`/`refactor:`/`test:`/`docs:`/`chore:`/`style:`, checked by lefthook: `make setup`), work on `develop`, the user pushes.
- The previous Vue/Express/Mongo app is at the `deprecated` tag.
- `AGENTS.md` files are links to these `CLAUDE.md`: edit the `CLAUDE.md`.
