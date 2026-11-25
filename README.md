# jat — job application tracker

Track job applications from a web app, a CLI and an MCP server for agents.

- `jat-server`: the HTTP API (Go, SQLite), the web app it serves, and the MCP over HTTP with its OAuth.
- `jat`: the CLI and the stdio MCP server, a client of the API only.
- `web/`: Nuxt 4 + Nuxt UI.

The previous Vue/Express/Mongo app is at the `deprecated` tag.

## Run

```sh
JAT_PASSWORD=change-me go run ./cmd/jat-server serve --db ./data/jat.db
```

## License

See [LICENSE](LICENSE).
