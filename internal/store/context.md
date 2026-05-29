# store — context

**Purpose:** The single typed gateway to Beacon's Postgres database (machines, jobs, audit).

**Public surface (so far):** the domain models — `Machine`, `Job`, `AuditEntry`, and the
`JobStatus`/`JobType` string enums (each with a `Valid()` method). Database operations
(`New`, `Migrate`, machine/job/audit methods) are added in later commits.

**Design / flow:** Pure typed models with `json` tags matching the DB column names.
Nullable timestamps are `*time.Time`; `payload`/`result`/`detail` are `json.RawMessage`.

**Depends on:** stdlib (`encoding/json`, `time`); `github.com/jackc/pgx/v5` once DB ops land.

**Extending it:** add new models/enums here with doc comments and `json` tags; cover enum
`Valid()` logic with table tests.
