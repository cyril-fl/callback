callback. Repo `cyril-fl/callback`, issues in French.

## Load to judge against

One user, a single account, a few hundred to a few thousand applications, the web app on a laptop or phone, the CLI and the MCP (an agent) as other clients. SQLite with a single connection.

## Where mistakes cost the most

1. **Auth** (`core/api/services/auth`, `guard`, `throttle`, `core/api/login`, `core/api/tokens`, `core/server/oauth`): session, API token and OAuth; what a non-session token may not do (manage tokens, log out, purge, delete from the trash for good); read-only tokens only GET and carry `Callback-Read-Only: true`.
2. **Batch writes** (`createapplications`, `updateapplications`, `deleteapplications`, `restoreapplications`, `forgetapplications`, `purgeapplications`, `importdata`): one transaction, valid items written and invalid ones in `failed`, an empty filter refused (`appfilter`).
3. **The single connection** (`core/services/sqlite`): a query on the `*bun.DB` while a `bun.Tx` is open hangs the server. Times stored in UTC (compared as text).
4. **Schema** (`createschema`, `migrations.go`): a NOT NULL column without default, a default on an `applications` column (bun drops it from batch inserts).
5. **Background job**: `markunanswered` (waiting applications moved to `no_reply`).
6. **The MCP** (`cli/cmd-mcp`, `core/server/mcphttp`): one tool call is one API request; the budget test.
7. **The web app** (`web/app`): `useApi()` only, no `v-html`, no `window.confirm`.
8. **Tests**: risky code above with no test that fails when it breaks.

## May run

`go vet ./...`, `make test` (in-memory SQLite, safe to run), `cd web && bun run typecheck`.
