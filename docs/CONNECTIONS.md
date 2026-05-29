# Beacon Connections & Setup (living doc)

Concrete wiring: what connects to what, in which direction, with which credentials, and
how to set it up. Updated alongside the code. _(Sections marked pending fill in as built.)_

## Connection map

| From | To | Direction | Transport | Auth |
|---|---|---|---|---|
| Laptop agent (`cmd/agent`) | Supabase | **outbound only** | HTTPS + Realtime (WebSocket, agent-initiated) | per-machine token |
| MCP server (`cmd/mcp`) | Supabase | outbound | HTTPS (REST/RPC) | service/Wingman token |
| Wingman (Emergent) | MCP server | inbound to MCP | MCP transport (stdio or HTTP) | Wingman token |

**Key property:** the laptop agent never accepts inbound connections — it dials out to
Supabase. So it works behind any home Wi-Fi / NAT / firewall with **no port forwarding**.

## Credentials & env

Configured via env (loaded by `internal/config`; never hardcoded, never committed — see
`.gitignore`). `.env.example` documents the keys. _(finalize when config lands)_

- `SUPABASE_URL` — project URL.
- `SUPABASE_ANON_KEY` / `SUPABASE_SERVICE_KEY` — keys (service key is server-side only).
- `BEACON_MACHINE_TOKEN` — per-machine token the agent presents.
- `BEACON_WINGMAN_TOKEN` — token the MCP server presents on Wingman's behalf.
- `BEACON_MACHINE_NAME` — human label for this laptop.

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

## Running locally _(pending)_

```bash
go build ./cmd/agent && ./beacon-agent     # on the laptop
go build ./cmd/mcp   && ./beacon-mcp        # in the cloud / locally for dev
```
