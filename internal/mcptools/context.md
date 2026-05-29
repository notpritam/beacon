# mcptools — context

**Purpose:** Translates MCP tool calls into typed store operations. Wraps the job queue
behind a small Go interface so the package is testable without a live server.

---

## Store interface

```go
type Store interface {
    ListMachines(ctx context.Context) ([]store.Machine, error)
    MachineByName(ctx context.Context, name string) (store.Machine, error)
    EnqueueJob(ctx context.Context, machineID string, jobType store.JobType,
               payload json.RawMessage, priority int, ttlAt *time.Time,
               createdBy string) (store.Job, error)
    GetJob(ctx context.Context, jobID string) (store.Job, error)
}
```

The real `*store.Store` satisfies this interface. Tests wire up a real DB instance
via `newToolsStore(t)` guarded by `TEST_DATABASE_URL`.

---

## Options

| Field             | Default  | Meaning                                              |
|-------------------|----------|------------------------------------------------------|
| `OnlineThreshold` | 30 s     | How recent a heartbeat must be to count as online.   |
| `PollInterval`    | 250 ms   | How often `enqueueAndWait` re-checks a running job.  |
| `WaitTimeout`     | 60 s     | Upper bound on how long a long-poll will block.      |

All fields default on `<= 0`, so `Options{}` is valid and picks up all defaults.

---

## Tools / New

`New(st Store, opts Options) *Tools` applies the defaults above and returns a `*Tools`
value. There is no teardown required — the store lifecycle is managed by the caller.

---

## The 7 tools

### Read-only (defined in `mcptools.go`)

| Tool              | Method signature                                           | What it does                             |
|-------------------|------------------------------------------------------------|------------------------------------------|
| `ListMachines`    | `(ctx) → ([]MachineInfo, error)`                           | All registered machines with online flag.|
| `MachineStatus`   | `(ctx, name) → (MachineInfo, error)`                       | Single machine by name.                  |
| `GetJob`          | `(ctx, jobID) → (JobOutcome, error)`                       | Current status + result of any job.      |

### Enqueuing (defined in `enqueue.go`)

| Tool         | Method signature                                                      | Store job type  |
|--------------|-----------------------------------------------------------------------|-----------------|
| `RunCommand` | `(ctx, machine, cmd, cwd string, timeoutSecs int) → (JobOutcome, error)` | `shell`      |
| `ReadFile`   | `(ctx, machine, path string) → (JobOutcome, error)`                   | `read_file`     |
| `WriteFile`  | `(ctx, machine, path, content string) → (JobOutcome, error)`          | `write_file`    |
| `ListDir`    | `(ctx, machine, path string) → (JobOutcome, error)`                   | `list_dir`      |

`screenshot` and `background` job types are not exposed here — the executor defers them.

---

## Online detection

`online(m store.Machine) bool` returns true when `m.LastSeen != nil` and
`time.Since(*m.LastSeen) < opts.OnlineThreshold`. A freshly registered machine with no
heartbeat is always considered offline.

---

## enqueueAndWait semantics

1. Calls `store.EnqueueJob` — always creates the job regardless of machine state.
2. If `!online(m)` — returns immediately with `{JobID: ..., Status: "queued", MachineOnline: false}`.
   The job parks in the queue and will be claimed when the agent next connects.
3. If `online(m)` — enters a long-poll loop:
   - Calls `store.GetJob` on each tick (`PollInterval`).
   - Returns as soon as the job reaches a terminal state (`done`, `failed`, or `expired`),
     with the result payload.
   - If `WaitTimeout` elapses before the job finishes, returns the current (non-terminal)
     status with `MachineOnline: true` so the caller can follow up via `GetJob`.
   - Respects `ctx.Done()` (returns `ctx.Err()` immediately).

**Note on `JobOutcome.MachineOnline`:** this field is populated by `enqueueAndWait` to
reflect whether the machine was online at enqueue time. `GetJob` does not have machine
context and always leaves `MachineOnline` false.

---

## Depends on

`github.com/notpritam/beacon/internal/store` (models, job types, job statuses).

## Extending it

- Add a new tool: add a method on `*Tools` in `enqueue.go` (or a new file), call
  `resolve` then `enqueueAndWait` with the appropriate `store.JobType`.
- If the `Store` interface needs a new method, add it to the interface in `mcptools.go`
  and update any test fakes.
- Cover every new code path with a DB-backed test in `*_test.go` guarded by
  `TEST_DATABASE_URL`.
