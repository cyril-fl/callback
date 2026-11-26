# web

Nuxt 4 (`ssr: false`), Nuxt UI 4 (Tailwind 4), `@nuxtjs/i18n` (fr only), VueUse, zod; the Go API on `runtimeConfig.public.apiBase` (`http://localhost:8080` in dev, same origin once generated and served by `callback-server serve --web`).

## Commands

`bun run dev`, `bun run generate`, `bun run test` (unit tests in `test/*.test.ts`, `node:test` under bun, on the pure code of `app/utils/`), `bun run typecheck`, `bun run lint` (ESLint, `@nuxt/eslint`), `bun run format` (Prettier), `bun run types` (API types from `../docs/swagger.yaml` into `app/types/api.d.ts`, committed).

Dev API: `CALLBACK_PASSWORD=... go run ./cmd/callback-server serve --db <file> --origin http://localhost:3000`.

## Rules

- API calls go through `useApi()`: cookies included, toast with the API's `error`, `/login` on a 401. Types are aliases of the generated `api.d.ts` in `app/types/items.ts`, never written by hand.
- Every text is in `i18n/locales/fr.json`; add `en.json` when English comes.
- Values are stored raw by the API: never `v-html`.
- Icon buttons carry an `aria-label`. Confirmations use a modal (`ConfirmModal`), never `window.confirm`.
- Icons come from `useIcons()`, keyboard shortcuts from `useShortcuts()`: never an `i-…` name or a key in a component.
- Use Nuxt UI components and tokens (`text-muted`, `bg-elevated`), not raw colors.
- Logic worth a test (computations, parsing, escaping) goes to `app/utils/` as plain functions, auto-imported by Nuxt and imported by the tests; pages keep the wiring.
- The API is the contract: if a change is needed there, report it, do not work around it here.
