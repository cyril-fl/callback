callback, SQLite, no migration tool.

- Settings are `domain.Preferences` (`core/domain/preferences.go`, `DefaultPreferences`), read and written by `readpreferences` / `updatepreferences` and the route `core/api/preferences` (GET/PATCH). The page is `web/app/pages/settings.vue`.
- `createschema` creates missing tables, indexes and the columns a model gained, at every start. A rename, a type change or data to rewrite is appended to `core/services/process/createschema/migrations.go` (append only). Run `callback-server backup <file>` first.
- A new NOT NULL column needs a default, **except on `applications`**: its columns have no SQL default (bun leaves a defaulted column out of a batch insert when the first row has its zero value); give the zero value meaning in Go instead.
- Errors: `domain.Invalid(...)` for a bad value (400).
- Then: `make test`, `make openapi` (OpenAPI and CLI client), `cd web && bun run types`, texts in `web/i18n/locales/fr.json` only (no English yet).
- No e2e: check the page with the `app-screenshot` skill.
