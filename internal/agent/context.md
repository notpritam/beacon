# agent — context

**Purpose:** Run the laptop side of Beacon: register the machine, drain its job queue,
execute each job, and report results back to the store.

**Public surface:**
- `Store` interface, `Executor` interface, `AuditMirror` interface, `Options`
- `Agent` — opaque struct; all fields unexported
- `New(st Store, exec Executor, audit AuditMirror, sentinel string, opts Options) *Agent`
- `(*Agent).Register(ctx, name, osName, token string) error`
- `(*Agent).RunOnce(ctx context.Context) (bool, error)`
- `(*Agent).Run(ctx context.Context) error`

**Design / flow:**

*New* wires together the store, executor, and local audit mirror without touching the
database. It applies sensible defaults: 2 s poll interval, 15 s heartbeat interval.
`sentinel` is a local file path for the local kill-switch signal; pass `""` to disable it.

*Register* hashes the raw token (SHA-256 hex) before persisting it so the plaintext
never lands in the DB, then arms the `killswitch.Checker` bound to this machine. All
subsequent loop calls require Register to have run first — otherwise they return
`errNotRegistered`.

*RunOnce* is a single-cycle attempt:
1. Kill-switch check (cloud flag via `MachineKillSwitch` + local sentinel file). If
   tripped → return `(false, nil)`.
2. Claim one job atomically (`ClaimNextJob`). If nothing queued → return `(false, nil)`.
3. `processJob`: call `StartJob`, run `Executor.Execute`, then call `CompleteJob` or
   `FailJob`. Append to cloud audit and local mirror after each state change (best-effort:
   audit failures are logged, never fatal).
4. If the context is cancelled mid-execution the job is left in `running` for the reaper
   (not marked failed). All other exec failures are recorded via `FailJob`.
5. Returns `(true, nil)` on success (even if the job itself failed), `(true, err)` on an
   infrastructure failure (claim/start/complete/fail DB error).

*Run* is the long-lived loop:
- Calls `beat` (heartbeat) immediately on entry, then again on every `HeartbeatInterval`
  tick.
- Fast-drains the queue by calling `RunOnce` in a tight loop as long as it returns
  `(true, nil)`.
- When the queue is empty (or RunOnce returns an error) it sleeps `PollInterval`, or wakes
  early on a heartbeat tick.
- On context cancellation Run returns `ctx.Err()` (e.g. `context.DeadlineExceeded`,
  `context.Canceled`).

**Interfaces (narrow for testability):**
- `Store` — 8 methods: `RegisterMachine`, `Heartbeat`, `ClaimNextJob`, `StartJob`,
  `CompleteJob`, `FailJob`, `AppendAudit`, `MachineKillSwitch`.
- `Executor` — `Execute(ctx, job) (json.RawMessage, error)`.
- `AuditMirror` — `Append(entry localaudit.Entry) error`.

`store.Store` satisfies `Store` directly; `executor.Executor` satisfies `Executor`;
`localaudit.Logger` satisfies `AuditMirror`.

**Depends on:** `internal/store` (types — `Machine`, `Job`), `internal/killswitch`,
`internal/localaudit`, stdlib (`crypto/sha256`, `encoding/hex`, `log/slog`, `time`).

**Extending it:** add new job-execution hooks by expanding the `Executor` interface;
add new stop signals by extending the `killswitch.Checker` (not this package).
