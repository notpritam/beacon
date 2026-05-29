# Beacon

Give a cloud agent safe, durable control of your laptop.

Beacon is an all-Go system that lets a cloud agent (**Wingman**) run shell commands,
read/write files, run background jobs, and take screenshots on your laptop through
**MCP tools** — routed via a **durable queue** so commands issued while the laptop is
offline are parked and run automatically on wake. Every action is audit-logged, and a
dual kill switch can halt the agent instantly.

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

Phase 0b complete: the laptop agent (`cmd/agent`) now drains the queue — it registers
the machine, polls for jobs, executes shell commands and file ops, reports results, and
stops cleanly on SIGINT/SIGTERM. The local audit mirror and dual kill switch are wired in.
The MCP server (`cmd/mcp`) is next (Phase 0c+).
Roadmap (background jobs → dashboard → interactive computer-use → fleet)
is in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

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

## License

MIT
