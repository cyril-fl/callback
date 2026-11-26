callback. No screenshot script: use the Chrome DevTools MCP.

- Stack: `CALLBACK_PASSWORD=change-me go run ./cmd/callback-server serve --db ./data/callback.db --origin http://localhost:3000 --mcp-addr :8081`, then `cd web && bun run dev` (app on :3000, API on :8080). Log in at `/login` with that password.
- Pages: `/` (applications), `/dashboard`, `/settings`, `/login`.
- The theme and the color mode stay on each device (`useTheme()`, `useColorMode()` in the browser's storage), not in `PATCH /api/preferences`: change them from `/settings` or the page's storage, and put them back afterwards. Account preferences (`PATCH /api/preferences`) are restored the same way.
- Never shoot against the user's real base: use `./data/callback.db` or a copy.
