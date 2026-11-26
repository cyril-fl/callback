callback.

- Branch `develop`; the user pushes.
- lefthook (`make setup`): gofmt with `stage_fixed` on Go files, Prettier `--check` on `web/` files, and the prefix check on the message.
- Build check after each commit: `go build ./... && go vet ./...`; for `web/`: `cd web && bun run typecheck`.
- Generated files follow their change: `docs/swagger.yaml` and `cli/services/apiclient/client.gen.go` (`make openapi`), `web/app/types/api.d.ts` (`bun run types` in `web/`).
