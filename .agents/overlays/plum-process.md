callback, plum v0.1.0.

- Processes live in `core/services/process/<verb><noun>/run.go`, one per business operation. References: `createapplications` (batch, valid items written, invalid ones in `failed`), `updatepreferences`, `createschema` (with `migrations.go`).
- The process takes a `bun.IDB`, so the API passes the base or a transaction and tests pass `testdb.New(t)`.
- Sentinel errors are made with `domain.Invalid/Unauthorized/Forbidden/NotFound` (`core/domain/errors.go`); `respond` maps them to 400/401/403/404, anything else is a 500 without detail.
- One connection: inside a transaction, use the `bun.Tx`, never the `*bun.DB`. Store times as `time.Now().UTC()`.
- A WHERE built from a `domain.Filter` goes through `core/services/appfilter`; an empty filter is refused.
