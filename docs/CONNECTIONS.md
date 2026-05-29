# Beacon Connections & Setup (living doc)

Concrete wiring: what connects to what, in which direction, with which credentials, and
how to set it up. Updated alongside the code. _(Sections marked pending fill in as built.)_

## Connection map

| From | To | Direction | Transport | Auth |
|---|---|---|---|---|
| Laptop agent (`cmd/agent`) | Supabase | **outbound only** | HTTPS + Realtime (WebSocket, agent-initiated) | per-machine token |
| MCP server (`cmd/mcp`) | Supabase | outbound | HTTPS (REST/RPC) | service/Wingman token |
| Wingman (Emergent) | MCP server | inbound to MCP | MCP streamable HTTP (`/mcp`) | `Authorization: Bearer <BEACON_WINGMAN_TOKEN>` |

**Key property:** the laptop agent never accepts inbound connections — it dials out to
Supabase. So it works behind any home Wi-Fi / NAT / firewall with **no port forwarding**.

## Credentials & env

Configured via env (loaded by `internal/config`; never hardcoded, never committed — see
`.gitignore`). `.env.example` documents the keys.

- `BEACON_DATABASE_URL` — **(required)** Postgres connection string (your Supabase DB URL
  or local Postgres). Everything fails without this.
- `BEACON_MACHINE_NAME` — human label for this machine (optional; defaults to hostname).
- `BEACON_MACHINE_TOKEN` — per-machine token the agent presents (optional in Phase 0;
  required when RLS / token checking is enforced in Phase 4).
- `BEACON_WINGMAN_TOKEN` — **(required for `cmd/mcp`)** Bearer token Wingman must send on
  every MCP request (`Authorization: Bearer <token>`). The MCP server refuses to start
  if this is unset. Keep this secret — it grants full remote-control access.
- `BEACON_MCP_ADDR` — TCP listen address for `cmd/mcp` (default `:8080`; e.g. `:9090`).

Supabase-specific keys (`SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_KEY`)
are deferred until the Supabase project is wired up.

## Supabase setup _(pending)_

1. Create a Supabase project.
2. Run the migration in `supabase/` to create `machines`, `jobs`, `audit_log` + the
   Storage bucket.
3. Enable Realtime on `jobs`.
4. Apply RLS policies (per-machine + Wingman scoping).
5. Issue tokens and register the machine.

## Token issuance & rotation _(pending)_
How tokens are minted, hashed at rest, presented, and rotated.

## Realtime channels _(pending)_
Which channel/filter the agent subscribes to for its machine's `queued` jobs, and the
poll-fallback interval + online threshold N.

## Local Postgres for tests

DB-backed tests read `TEST_DATABASE_URL` and **skip** when it is unset (so the gate stays
green without a database). To run them, point it at any Postgres. A throwaway one via Docker:

```bash
docker run --rm -e POSTGRES_PASSWORD=beacon -p 5433:5432 postgres:16
export TEST_DATABASE_URL=postgres://postgres:beacon@localhost:5433/postgres
go test ./... -p 1
```

DB-backed tests share one database; run the full suite with `-p 1` to serialize packages
(otherwise the store package's `TRUNCATE` can collide with another package's rows).

(Any local Postgres works — e.g. a Homebrew `postgresql@16` instance with a `beacon_test`
database; set `TEST_DATABASE_URL` to its connection string.)

## Running locally

```bash
export BEACON_DATABASE_URL=postgres://...        # your Postgres / Supabase DB URL
go run ./cmd/beaconctl migrate                   # create the schema
go run ./cmd/beaconctl machines                  # list registered machines
```

```bash
# start the MCP server (Phase 0c)
BEACON_DATABASE_URL=postgres://... BEACON_WINGMAN_TOKEN=<secret> go run ./cmd/mcp
```
