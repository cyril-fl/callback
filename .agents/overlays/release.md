callback.

- Repo `cyril-fl/callback`, branch `develop` (not the default branch: `Fixes #n` closes nothing, close by hand). Tags `vX.Y.Z` (last: see `git tag --sort=-v:refname | head -1`).
- Checks: `make test`; when a commit touches `web/`: `cd web && bun run test && bun run typecheck && bun run lint`. No e2e suite.
- READMEs: `README.md` (server, CLI, API, web app), `MCP.md` (in French) when an MCP tool changes.
- Issues and their comments in French.
- No handoff file.
- The user runs `git push origin develop --follow-tags`.
- Hotfix (the user says "hotfix"): no checks, no subagent, no issues. Commit, annotated tag, then push yourself: `git push origin develop --follow-tags`.
