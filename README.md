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

Phase 0a (data layer) complete: config, pgx-backed store with atomic-claim queue,
migrations, audit log, kill switch, and the `beaconctl` admin CLI are all implemented.
The laptop agent (`cmd/agent`) and MCP server (`cmd/mcp`) are next (Phase 0b+).
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

# run tests (DB-backed tests skip if TEST_DATABASE_URL is unset)
TEST_DATABASE_URL=postgres://postgres:beacon@localhost:5433/postgres go test ./...

# migrate a database and inspect state
BEACON_DATABASE_URL=postgres://postgres:beacon@localhost:5433/postgres go run ./cmd/beaconctl migrate
BEACON_DATABASE_URL=postgres://postgres:beacon@localhost:5433/postgres go run ./cmd/beaconctl machines
```

## License

MIT
