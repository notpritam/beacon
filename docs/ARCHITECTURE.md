# Beacon Architecture (living doc)

This is the canonical doc for how Beacon is **built** and how a command flows
end-to-end. It is updated in the same commit as the code it describes.

> Status: Computer-use live end-to-end. Wingman can call all **9 tools** over streamable HTTP with bearer auth. The laptop agent claims, executes, and reports jobs (polling) — shell, file ops, screenshot, and `gui` computer-use (macOS); background jobs deferred. An optional token-gated live-view dashboard streams the screen to a browser on demand.

## Components

| Component | Path | Runs on | Role |
|---|---|---|---|
| Laptop agent | `cmd/agent` | the laptop | The hands: claims jobs, executes, reports. Outbound-only. |
| MCP server | `cmd/mcp` | cloud | The translator: turns Wingman tool calls into jobs, returns results. |
| Coordinator | Supabase | cloud | The mailbox + ledger: queue, machines, audit, Storage, realtime, auth. |

Internal slices (each has its own `context.md`):

- **Implemented (Phase 0a):** `internal/config`, `internal/store`
- **Implemented (Phase 0b):** `internal/executor`, `internal/killswitch`, `internal/localaudit`, `internal/agent`
- **Implemented (Phase 0c):** `internal/mcptools`, `cmd/mcp` — 9 tools over streamable HTTP with bearer auth (screenshot + `gui` computer-use macOS-only; background jobs deferred)
- **Implemented (live view):** `internal/dashboard` — opt-in, token-gated, on-demand MJPEG screen stream + jobs feed, served by `cmd/agent`
- **Pending (later phases):** `internal/queue`

Admin/debug CLI: `cmd/beaconctl` — `migrate`, `machines`, `enqueue <machine> <cmd>`, `get <job-id>`.

Laptop daemon: `cmd/agent` — registers the machine and drains its job queue (shell, file ops, screenshot, `gui` computer-use); optionally serves the live-view dashboard; clean shutdown on SIGINT/SIGTERM.

MCP server: `cmd/mcp` — serves the 9 Beacon tools over streamable HTTP at `/mcp`; requires `Authorization: Bearer <BEACON_WINGMAN_TOKEN>` on every request.

Live-view dashboard: `internal/dashboard` — served by `cmd/agent` when `BEACON_DASHBOARD_TOKEN` is set. Token-gated routes `/` (page), `/live` (MJPEG stream), `/jobs` (feed). Capture is per-request, so the screen is only grabbed while a viewer is connected. Observation only — it does not replace the `screenshot` job the agent uses to see the screen.

## End-to-end flow

```
You ─talk─▶ Wingman (cloud) ─MCP tool─▶ cmd/mcp ─insert queued job─▶ Supabase
                                                                       │
                                            cmd/agent ◀──poll (0b)─────┘
                                            claims (atomic) ▶ executes ▶ writes result
                                                                       │
                            Wingman ◀─returns result─ cmd/mcp ◀─long-poll job row─┘
```

1. Wingman calls an MCP tool (e.g. `run_command(machine, cmd)`).
2. `cmd/mcp` inserts a `jobs` row, `status=queued`, scoped to the target machine.
3. **Laptop awake?** decided by the agent's heartbeat (`machines.last_seen`).
   - Online → agent picks the row up on its next poll (Phase 0b; realtime push later), claims, runs.
   - Offline → row stays `queued` (until `ttl_at`); MCP returns `{job_id, status:queued}`.
4. Agent claims atomically (`queued→claimed`), executes, writes `result` + `status=done`.
   Shell, file ops, screenshot, and `gui` computer-use (macOS) are implemented; background
   jobs are deferred.
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

One index: **`idx_jobs_claimable`** on `(machine_id, status, priority DESC, created_at)`,
used by the `FOR UPDATE SKIP LOCKED` claim path.

Screenshots are returned inline as a downscaled base64 JPEG in the job result; a Storage
bucket for very large output is deferred to a later phase.

## Security model

Per-machine token + Wingman token (hashed), Supabase RLS scoping rows, local + cloud
audit log, dual kill switch (`machines.kill_switch` + local sentinel). Detail in
`docs/CONNECTIONS.md` and design spec §7.

## Deviations from the spec

- **Data access via `pgx` direct, not the Supabase SDK** (spec §3.2). Supabase is Postgres;
  this gives atomic `FOR UPDATE SKIP LOCKED` claims and transactions.
- **RLS deferred to Phase 4** (spec §7). Phase 0 auth = connection-string secret +
  per-machine token checked in app logic.
- **Phase 0b ships pure polling** — the agent polls for claimable jobs on an interval.
  Realtime push (Postgres `LISTEN/NOTIFY`, not Supabase Realtime) is deferred to a later phase.

### Known deferrals (later phases)

- A `schema_migrations` version table + multi-file migration atomicity, before the first
  non-idempotent migration.
- A lease/reaper for jobs stuck in `claimed`/`running` (e.g. a claimer that crashes);
  `ExpireDueJobs` currently only expires `queued` jobs past their TTL.
- State-transition validation in `setStatus` (currently any status→any status), to land
  with the agent execution loop.
- Indexes on `audit_log(job_id, machine_id)` if/when audit is queried by those.
- Heartbeat during a long job: execution is synchronous, so a job running longer than the
  heartbeat interval delays the next heartbeat (the machine can look briefly stale). A
  background-goroutine heartbeat would decouple this.
- The kill switch is a **pre-claim gate** (checked before claiming a job); it does not
  abort an already-running job. Mid-job abort would come with the reaper/lease work.
