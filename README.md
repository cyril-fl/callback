# callback — job application tracker

Track job applications from a web app, a CLI and an MCP server for AI agents. Single user, one SQLite file.

- **`callback-server`**: the HTTP API (Go, SQLite), the web app it serves, and the MCP over HTTP with its OAuth sign-in.
- **`callback`**: the CLI and the stdio MCP server, a client of the API only.
- **`web/`**: Nuxt 4 + Nuxt UI: a dashboard with a sortable, filterable table or a kanban by status, the add/edit
  form, API tokens, four themes and dark mode.

Each application has a company, an address (location is the city, plus an optional street, postal code and country),
a website, a source (spontaneous, job post with its URL, recruiter with their e-mail, or any name you give), a status
(applied → responded → interview → offer / rejected), a rating out of 5, a comment, the day it was sent, and optionally
the interview's date and time and a contact (name, e-mail). Every write is a batch: one request adds, changes or deletes many applications in one transaction.

The previous Vue/Express/Mongo app is at the `deprecated` tag.

## Deploy on your server (Docker)

The image is `ghcr.io/cyril-fl/callback`: one static binary on `scratch`, the web app included, the base in the `/data`
volume. Pushing a `v*` tag builds and publishes it (`:<version>` and `:latest`, amd64 and arm64).

1. On the server, copy `compose.prod.yaml` and `.env.example` (as `.env`) into a folder, and fill `.env`:
   - `CALLBACK_PASSWORD`: the password, 8 characters or more. It is read only on the first start, when the base is
     empty; change it later with `docker compose -f compose.prod.yaml exec callback /callback-server --db /data/callback.db reset-password`.
   - `CALLBACK_ORIGIN`: the address you open in the browser, e.g. `https://callback.<tailnet>.ts.net`.
2. If the GitHub package is private: `docker login ghcr.io` with a token that can read packages.
3. `docker compose -f compose.prod.yaml up -d`. Update: `docker compose -f compose.prod.yaml pull && docker compose -f compose.prod.yaml up -d`.

Both ports listen on 127.0.0.1 only (`CALLBACK_LISTEN`, `CALLBACK_MCP_LISTEN` change them):

- **8080, the API and the web app**: keep it private, e.g. `tailscale serve --bg --https=443 http://127.0.0.1:8080`.
- **8081, the MCP and its OAuth**: the only port to open to the Internet, for AI apps that call it from their own
  servers: `tailscale funnel --bg --https=8443 http://127.0.0.1:8081`, or a Cloudflare tunnel to `127.0.0.1:8081`.
  Wrong passwords on its sign-in page make the caller wait, per address and across all addresses.

Backup (the image has no sqlite3; `VACUUM INTO` copies the base while it runs):

```sh
docker compose -f compose.prod.yaml exec callback /callback-server --db /data/callback.db backup /data/backup-$(date +%F).db
docker cp callback:/data/backup-$(date +%F).db .
```

## The CLI

```sh
go install github.com/cyril-fl/callback/cmd/callback@latest
callback login https://callback.<tailnet>.ts.net        # asks the password (or CALLBACK_PASSWORD)
callback add --company Acme --location Paris --country France --source job_post --source-detail https://jobs.acme.example/42
callback add --file applications.json             # a JSON array, or - for stdin
callback ls --status interview,offer -q acme      # --json for scripts
callback edit <id> --status interview             # or by filters: --where-status applied --untouched-since 2026-01-01 --dry-run
callback rm <id>...
callback stats
callback token add claude --read-only --expires 90d
```

The session is kept in `~/.config/callback/session.json`; `CALLBACK_URL` and `CALLBACK_SESSION` override it.

## AI agents (MCP)

Six tools, all batch: `list_applications`, `read_applications`, `add_applications`, `update_applications`,
`delete_applications`, `application_stats`. A read-only token gets the read tools only.

- Local: `claude mcp add callback -- callback mcp` (uses the CLI's session).
- Remote: `claude mcp add --transport http callback https://<your MCP address>/mcp`, signed in through OAuth, or with
  `--header "Authorization: Bearer callback_…"` and a token from the settings or `callback token add`.

Details, in French: [MCP.md](MCP.md).

## Develop

```sh
make setup                                    # git hooks (gofmt, prettier, conventional commits)
CALLBACK_PASSWORD=change-me go run ./cmd/callback-server serve --db ./data/callback.db --origin http://localhost:3000 --mcp-addr :8081
make web                                      # Nuxt on http://localhost:3000
make test                                     # go vet + go test, each test on an in-memory SQLite
make openapi                                  # after an API change, then `bun run types` in web/
make up / make down                           # the Docker stack, built locally
```

Layout and rules for contributors and agents: [CLAUDE.md](CLAUDE.md).

## License

See [LICENSE](LICENSE).
