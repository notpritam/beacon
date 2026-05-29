# store — context

**Purpose:** The single typed gateway to Beacon's Postgres database (machines, jobs, audit).

**Public surface:**
- `Machine`, `Job`, `AuditEntry` — domain models with `json` tags matching DB column names.
- `JobStatus`, `JobType` — string enums with `Valid()` methods.
- `New(ctx, databaseURL) (*Store, error)` — opens and pings a `pgxpool.Pool`.
- `(*Store).Close()` — releases the pool.
- `(*Store).Migrate(ctx) error` — applies all embedded SQL migrations in `migrations/`
  in lexical filename order. Migrations are idempotent (`CREATE ... IF NOT EXISTS`).

**Design / flow:**
- `store.go` owns the `Store` struct and connection lifecycle.
- `migrate.go` embeds `migrations/*.sql` via `//go:embed` and runs them via `pool.Exec`.
  Single `0001_init.sql` creates `machines`, `jobs`, and `audit_log` tables plus the
  `idx_jobs_claimable` index.
- Nullable timestamps are `*time.Time`; `payload`/`result`/`detail` are `json.RawMessage`.

**Depends on:** `github.com/jackc/pgx/v5/pgxpool`; stdlib `embed`, `sort`, `context`.

**Extending it:**
- Add new SQL migrations as `0002_*.sql`, `0003_*.sql`, etc. — they run automatically
  in filename order on the next `Migrate` call.
- Add machine/job/audit query methods to `store.go` or new files in this package.
- The `newTestStore` test helper in `storetest_test.go` connects, migrates, and truncates;
  all DB-backed tests should call it and rely on its skip logic.
