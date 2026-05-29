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

Full design: [`docs/superpowers/specs/2026-05-29-beacon-design.md`](docs/superpowers/specs/2026-05-29-beacon-design.md).

## Status

Phase 0 (MVP) in progress: schema, agent, MCP server, auth, audit, kill switch on one
macOS machine. Roadmap (background jobs → dashboard → interactive computer-use → fleet)
is in the design spec.

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
[`CLAUDE.md`](CLAUDE.md) for the engineering rules.

## License

MIT
