# Beacon Architecture (living doc)

This tracks how Beacon is **actually built** and how a command flows end-to-end. It is
updated in the same commit as the code it describes. The point-in-time design rationale
lives in `docs/superpowers/specs/2026-05-29-beacon-design.md`; deviations from it are
noted here.

> Status: Phase 0 (MVP) — scaffolding. Sections marked _(pending)_ fill in as code lands.

## Components

| Component | Path | Runs on | Role |
|---|---|---|---|
| Laptop agent | `cmd/agent` | the laptop | The hands: claims jobs, executes, reports. Outbound-only. |
| MCP server | `cmd/mcp` | cloud | The translator: turns Wingman tool calls into jobs, returns results. |
| Coordinator | Supabase | cloud | The mailbox + ledger: queue, machines, audit, Storage, realtime, auth. |

Internal slices (each has its own `context.md`): `internal/config`, `internal/supabase`,
`internal/queue`, `internal/executor`, `internal/audit`, `internal/killswitch`,
`internal/mcptools`. _(created as implemented)_

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
Atomic claim: `UPDATE jobs SET status='claimed', claimed_at=now() WHERE id=$1 AND status='queued' RETURNING *`. Only one claimant wins → no double execution. _(impl pending)_

## Data model

`machines`, `jobs`, `audit_log` + a Storage bucket for large output/screenshots. Full
column list in the design spec §4; the SQL migration lives in `supabase/` _(pending)_.

## Security model

Per-machine token + Wingman token (hashed), Supabase RLS scoping rows, local + cloud
audit log, dual kill switch (`machines.kill_switch` + local sentinel). Detail in
`docs/CONNECTIONS.md` and design spec §7.

## Deviations from the spec

_None yet._
