# beaconagent (cmd/agent) — context

**Purpose:** The laptop daemon — registers the machine and drains its job queue.

**Public surface:** the `beacon-agent` binary. Env: `BEACON_DATABASE_URL` (required),
`BEACON_MACHINE_NAME` (defaults to hostname), `BEACON_MACHINE_TOKEN`.

**Design / flow:** `config.Load` -> `store.New` -> `executor.New` -> `localaudit.New` ->
`agent.New` -> `Register` -> `Run`, under a `signal.NotifyContext` (SIGINT/SIGTERM) for
clean shutdown. State lives under `~/.beacon/`: `audit.log` (local mirror) and `killswitch`
(create/touch this file to hard-stop the agent locally).

**Depends on:** `internal/{config,store,executor,localaudit,agent}`.

**Extending it:** flags/config for poll & heartbeat intervals; launchd packaging (Phase 0d).
