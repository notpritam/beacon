# Beacon

Give a cloud agent safe, durable control of your laptop.

Beacon is an all-Go system that lets a cloud agent (**Wingman**) run shell commands,
read/write files, take screenshots, and drive the GUI (move/click, type, hotkeys,
open/quit apps — full computer-use) on your laptop through **MCP tools** — routed via a
**durable queue** so commands issued while the laptop is offline are parked and run
automatically on wake. An optional, token-gated **live-view dashboard** lets you watch
the screen from a browser. Every action is audit-logged, and a dual kill switch (cloud
flag + local sentinel) stops the agent from claiming new jobs.

> **Note:** *Beacon* is the project. *Wingman* is the cloud agent that uses it.

## How it works

```
You → Wingman → MCP server → Supabase queue → Laptop agent → executes → result flows back
```

- **`cmd/agent`** — the laptop daemon (single static Go binary, low RAM, outbound-only).
- **`cmd/mcp`** — the cloud MCP server Wingman talks to.
- **Supabase** — Postgres queue + machines + audit log, Storage, realtime, RLS auth.

Full design and flow: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Status

Computer-use is live end-to-end. The MCP server (`cmd/mcp`) exposes **9 tools** over
streamable HTTP with bearer auth: `run_command`, `read_file`, `write_file`, `list_dir`,
`screenshot`, `gui` (mouse/keyboard/apps), `get_job`, `list_machines`, `machine_status`.
The laptop agent (`cmd/agent`) drains the queue — registers the machine, polls for jobs,
executes shell/file/screenshot/GUI jobs, reports results, and stops cleanly on
SIGINT/SIGTERM. An optional live-view dashboard (`internal/dashboard`, gated by
`BEACON_DASHBOARD_TOKEN`) streams the screen to a browser on demand. The local audit
mirror and dual kill switch are wired in. Roadmap (background jobs → fleet) is in
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Development

Requires Go 1.25+.

```bash
# one-time: enable the commit quality gate
git config core.hooksPath .githooks

# linter used by the gate
brew install golangci-lint        # or: see https://golangci-lint.run/welcome/install/
```

The pre-commit gate runs gofmt, `go vet`, `golangci-lint`, build, tests, and the
ABOUTME-header check on staged Go files. See [`CONVENTIONS.md`](CONVENTIONS.md) and
[`docs/GO_STYLE.md`](docs/GO_STYLE.md) for the engineering rules.

### Data layer (Phase 0a)

```bash
# start a throwaway Postgres for tests
docker run --rm -e POSTGRES_PASSWORD=beacon -p 5433:5432 postgres:16

# run tests (DB-backed tests skip if TEST_DATABASE_URL is unset).
# DB-backed tests share one database; run the full suite with -p 1 to serialize packages.
TEST_DATABASE_URL=postgres://postgres:beacon@localhost:5433/postgres go test ./... -p 1

# migrate a database and inspect state
BEACON_DATABASE_URL=postgres://postgres:beacon@localhost:5433/postgres go run ./cmd/beaconctl migrate
BEACON_DATABASE_URL=postgres://postgres:beacon@localhost:5433/postgres go run ./cmd/beaconctl machines
```

### Laptop agent (Phase 0b)

```bash
# registers, migrates the schema, and drains the queue
BEACON_DATABASE_URL=<db-url> BEACON_MACHINE_TOKEN=<token> go run ./cmd/agent

# optional: set a human name (defaults to hostname)
BEACON_DATABASE_URL=<db-url> BEACON_MACHINE_NAME=my-laptop BEACON_MACHINE_TOKEN=<token> go run ./cmd/agent
```

State files are written to `~/.beacon/`: `audit.log` (local append-only mirror) and
`killswitch` (create/touch to hard-stop the agent without a signal).

### MCP server (Phase 0c)

```bash
# serve Beacon's 9 tools over streamable HTTP at /mcp
BEACON_DATABASE_URL=<db-url> BEACON_WINGMAN_TOKEN=<wingman-secret> go run ./cmd/mcp

# optional: override the listen address (default :8080)
BEACON_DATABASE_URL=<db-url> BEACON_WINGMAN_TOKEN=<wingman-secret> BEACON_MCP_ADDR=:9090 go run ./cmd/mcp
```

Wingman connects to `http://<host>:<port>/mcp` and must send
`Authorization: Bearer <BEACON_WINGMAN_TOKEN>` on every request.

### Live-view dashboard (optional)

The agent can serve a browser live view — an MJPEG screen stream plus a recent-jobs feed
— so you can *watch* what Wingman is doing. It is **opt-in** (off unless a token is set)
and **on-demand** (the screen is only captured while a viewer is connected).

```bash
# enable by setting a token; the agent then serves the dashboard on :8081
BEACON_DASHBOARD_TOKEN=<view-secret> BEACON_DASHBOARD_ADDR=:8081 \
  BEACON_DATABASE_URL=<db-url> BEACON_MACHINE_TOKEN=<token> go run ./cmd/agent
```

Open `http://localhost:8081/?token=<BEACON_DASHBOARD_TOKEN>`. Every route is token-gated.
To reach it remotely, expose `:8081` over a tunnel (e.g. `cloudflared tunnel --url
http://localhost:8081`) separate from the MCP tunnel.

## License

MIT
