# beaconctl — context

**Purpose:** Admin/debug CLI over the `store` package: run migrations, list machines,
enqueue a shell job, fetch a job.

**Public surface:** the `beaconctl <command>` binary. Commands: `migrate`,
`machines`, `enqueue <machine> <cmd>`, `get <job-id>`.

**Design / flow:** thin wrapper — `config.Load` → `store.New` (under a 30s timeout
context so a hung connect can't block forever) → dispatch on `args[0]`. No logic lives
here; it only calls `store`.

**Depends on:** `internal/config`, `internal/store`.

**Extending it:** add a `case` per new command; keep it a thin shell over `store`.
