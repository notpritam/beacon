# Beacon Architecture (living doc)

This is the canonical doc for how Beacon is **built** and how a command flows
end-to-end. It is updated in the same commit as the code it describes.

> Status: Phase 0 (MVP) — scaffolding. Sections marked _(pending)_ fill in as code lands.

## Components

| Component | Path | Runs on | Role |
|---|---|---|---|
| Laptop agent | `cmd/agent` | the laptop | The hands: claims jobs, executes, reports. Outbound-only. |
| MCP server | `cmd/mcp` | cloud | The translator: turns Wingman tool calls into jobs, returns results. |
| Coordinator | Supabase | cloud | The mailbox + ledger: queue, machines, audit, Storage, realtime, auth. |

Internal slices (each has its own `context.md`):

- **Implemented (Phase 0a):** `internal/config`, `internal/store`
- **Pending (later phases):** `internal/queue`, `internal/executor`, `internal/audit`,
  `internal/killswitch`, `internal/mcptools`

Admin/debug CLI: `cmd/beaconctl` — `migrate`, `machines`, `enqueue <machine> <cmd>`, `get <job-id>`.

## End-to-end flow

```
You ─talk─▶ Wingman (cloud) ─MCP tool─▶ cmd/mcp ─insert queued job─▶ Supabase
                                                                       │
                                            cmd/agent ◀─realtime/poll──┘
                                            claims (atomic) ▶ executes ▶ writes result
                                                                       │
                            Wingman ◀─returns result─ cmd/mcp ◀─long-poll job row─┘
```

1. Wingman calls an MCP tool (e.g. `run_command(machine, cmd)`).
2. `cmd/mcp` inserts a `jobs` row, `status=queued`, scoped to the target machine.
3. **Laptop awake?** decided by the agent's heartbeat (`machines.last_seen`).
   - Online → agent gets the row via realtime (sub-second), claims, runs.
   - Offline → row stays `queued` (until `ttl_at`); MCP returns `{job_id, status:queued}`.
4. Agent claims atomically (`queued→claimed`), executes, writes `result` + `status=done`.
5. `cmd/mcp` long-polls the row and returns the result to Wingman.

## Job lifecycle

```
queued ─claim─▶ claimed ─start─▶ running ─▶ done | failed
   └─ ttl_at passes ─▶ expired
```
Atomic claim: `UPDATE jobs SET status='claimed', claimed_at=now() WHERE id=$1 AND status='queued' RETURNING *`. Only one claimant wins → no double execution. Implemented and concurrency-tested in `internal/store`.

## Data model

The schema is implemented. Source of truth: `internal/store/migrations/0001_init.sql`.

Three tables:

- **`machines`** — registered agents: name, token (hashed), heartbeat, kill-switch flag.
- **`jobs`** — the durable queue: status, payload, result, TTL, claimed/completed timestamps.
- **`audit_log`** — append-only record of every status transition and action.

One index: **`idx_jobs_claimable`** on `(machine_name, status, created_at)`, used by
the `FOR UPDATE SKIP LOCKED` claim path.

A Storage bucket for large output/screenshots is deferred to a later phase.

## Security model

Per-machine token + Wingman token (hashed), Supabase RLS scoping rows, local + cloud
audit log, dual kill switch (`machines.kill_switch` + local sentinel). Detail in
`docs/CONNECTIONS.md` and design spec §7.

## Deviations from the spec

- **Data access via `pgx` direct, not the Supabase SDK** (spec §3.2). Supabase is Postgres;
  this gives atomic `FOR UPDATE SKIP LOCKED` claims and transactions.
- **RLS deferred to Phase 4** (spec §7). Phase 0 auth = connection-string secret +
  per-machine token checked in app logic.
- **Realtime via Postgres `LISTEN/NOTIFY`** (Phase 0b), not Supabase Realtime.

### Known deferrals (later phases)

- A `schema_migrations` version table + multi-file migration atomicity, before the first
  non-idempotent migration.
- A lease/reaper for jobs stuck in `claimed`/`running` (e.g. a claimer that crashes);
  `ExpireDueJobs` currently only expires `queued` jobs past their TTL.
- State-transition validation in `setStatus` (currently any status→any status), to land
  with the agent execution loop.
- Indexes on `audit_log(job_id, machine_id)` if/when audit is queried by those.
