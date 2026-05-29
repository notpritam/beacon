# store — context

**Purpose:** The single typed gateway to Beacon's Postgres database (machines, jobs, audit).

**Public surface:**
- Lifecycle: `New(ctx, url)`, `(*Store).Close()`, `(*Store).Migrate(ctx)`. `ErrNotFound`
  is returned by lookups/updates when the target row does not exist.
- Machines: `RegisterMachine` (idempotent upsert on name), `MachineByName`, `ListMachines`
  (all machines, newest first), `Heartbeat`, `SetKillSwitch`, `MachineKillSwitch` (read
  the current kill-switch state by machine ID).
- Jobs: `EnqueueJob`, `ClaimNextJob` (atomic, returns nil when the queue is empty),
  `StartJob`, `CompleteJob`, `FailJob`, `GetJob`, `ExpireDueJobs`.
- Audit: `AppendAudit` (append-only), `ListAudit` (newest first).
- Models: `Machine`, `Job`, `AuditEntry`, `JobStatus`, `JobType` (each enum has `Valid()`).

**Design / flow:** Wraps a `pgxpool.Pool`. The queue claim uses
`FOR UPDATE SKIP LOCKED` so concurrent claimers never get the same job (verified by a
concurrent test). Lifecycle updates (`setStatus`) and lookups map a missing row to
`ErrNotFound`. Migrations are embedded SQL applied in filename order, idempotent via
`IF NOT EXISTS`. Structs carry `json` tags matching the DB column names; nullable
timestamps are `*time.Time`; `payload`/`result`/`detail` are `json.RawMessage`.
`audit_log` has no foreign keys and nullable `job_id`/`machine_id`, so the append-only
trail survives deletion of the referenced job/machine — a compromised cloud row can't
erase history. RLS is not used in Phase 0 — the connection string is the secret and
per-machine tokens are checked in app logic.

**Depends on:** `github.com/jackc/pgx/v5` (+ `pgxpool`), Postgres.

**Extending it:** add a new `NNNN_*.sql` migration for schema changes; add one method per
operation with SQL in a `const` next to it; cover every new query with a DB-backed test
guarded by `TEST_DATABASE_URL`. Future (later phases): a `schema_migrations` version
table and multi-file migration atomicity once a non-idempotent migration is needed; a
lease/reaper for jobs stuck in `claimed`/`running`; state-transition validation in
`setStatus`.
