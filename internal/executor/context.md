# executor — context

**Purpose:** runs a `store.Job` on the local machine and returns its result as JSON.

**Public surface:**
- `Config` / `DefaultConfig()` — execution limits (timeouts, output caps).
- `Executor` / `New(cfg Config) *Executor` — the runner.
- `(*Executor).Execute(ctx, job) (json.RawMessage, error)` — dispatch a job and get back its JSON result.

**Design / flow:**
- `Execute` dispatches on `job.Type` using an `if` chain (single type in Task 1; will become a switch in Task 2 when more cases are added).
- Shell jobs (`JobShell`) are handled by `runShell` in `shell.go`:
  1. Unmarshal the payload into `shellPayload{Cmd, Cwd, TimeoutSecs}`.
  2. Compute the effective timeout (payload overrides default; capped at MaxTimeout).
  3. Run `sh -c <cmd>` via `exec.CommandContext` with the derived timeout context.
  4. On timeout (`runCtx.Err() != nil`) return a wrapped context error — this is an execution error.
  5. On non-zero exit (`exec.ExitError`) record the exit code in `shellResult` — NOT an error return.
  6. Marshal and return `shellResult{Stdout, Stderr, ExitCode}`.
- Output is capped per-stream at `MaxOutputBytes` via `capString`.
- Unsupported job types return an explicit `fmt.Errorf` (not a panic).

**Depends on:** `internal/store` (for `Job`, `JobType` constants). No database connection needed.

**Extending it:**
- Add new job types as new `if job.Type == store.JobXxx` branches (Task 2 converts to switch).
- Keep each operation in its own file (`shell.go`, `files.go`, etc.) mirroring the job type.
- Tests are pure — use `t.TempDir()` and real shell; no mocks or DB needed.
