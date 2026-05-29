# agent — context

**Purpose:** Run the laptop side of Beacon: register the machine, drain its job queue,
execute each job, and report results back to the store.

**Public surface:** `Store` interface, `Executor` interface, `AuditMirror` interface,
`Options`, `Agent`, `New(st, exec, audit, sentinel, opts) *Agent`,
`(*Agent).Register(ctx, name, osName, token) error`.

**Design / flow:**
- `New` wires together the store, executor, and local audit mirror without touching the
  database; it applies sensible defaults for poll/heartbeat intervals.
- `Register` hashes the raw token (SHA-256 hex) before persisting it so the plaintext
  never lands in the DB, then arms the `killswitch.Checker` bound to this machine.
- The three interfaces (`Store`, `Executor`, `AuditMirror`) are narrow so tests can
  substitute fakes without a real DB or filesystem. `store.Store` satisfies `Store`
  directly; `executor.Executor` satisfies `Executor`; `localaudit.Logger` satisfies
  `AuditMirror`.
- The run loop (`RunOnce`, `Run`) lives in Task 2 and uses the fields wired here.

**Depends on:** `internal/store` (types only — `Machine`, `Job`), `internal/killswitch`,
`internal/localaudit`, stdlib (`crypto/sha256`, `encoding/hex`, `log/slog`, `time`).

**Extending it:** add new job-execution hooks by expanding the `Executor` interface;
add new stop signals by extending the `killswitch.Checker` (not this package).
