# Beacon Engineering Conventions

These are the rules that keep Beacon clean as it grows. They are enforced by the
pre-commit gate where possible and by review otherwise. They override default habits.

## 1. Project layout (idiomatic Go, feature-sliced internals)

Go doesn't use frontend "FSD", but we apply the same spirit: **one folder = one slice
with a clear boundary, documented in a `context.md`.**

```
beacon/
  cmd/
    agent/        # entrypoint: the laptop daemon
    mcp/          # entrypoint: the cloud MCP server
  internal/       # all private packages — one concern each
    config/       # env/config loading + validation
    supabase/     # typed Supabase client (jobs, machines, audit, storage, realtime)
    queue/        # job lifecycle: claim, transition, ttl
    executor/     # run shell / file ops / screenshot
    audit/        # local append-only audit mirror
    killswitch/   # cloud-flag + local-sentinel kill switch
    mcptools/     # MCP tool definitions wired by cmd/mcp
  docs/
    superpowers/specs/   # design specs
  scripts/        # dev/CI scripts (e.g. check-aboutme.sh)
  .githooks/      # pre-commit gate
```

- `cmd/*` packages are thin: parse config, wire dependencies, run. Logic lives in `internal/*`.
- A package depends on its dependencies through small interfaces it declares, so each
  slice is testable in isolation. Don't reach across slices via globals.
- When a package starts doing two things, split it. A growing file is a signal.

## 2. Every file: `// ABOUTME:` header (enforced)

The first two comment lines of every `.go` file (after any `//go:build` constraint)
must be:

```go
// ABOUTME: <what this file is>
// ABOUTME: <the one extra thing worth knowing>
```

The pre-commit hook rejects any staged `.go` file missing both lines.

## 3. Every folder: `context.md` (the "about this slice" doc)

Each package folder under `cmd/` and `internal/` has a `context.md`. It is the design
memory of that slice so a future change (by you or an agent) stays consistent. Keep it
short and current — update it in the same commit that changes the slice.

Template:

```markdown
# <package> — context

**Purpose:** one sentence — what this slice owns.

**Public surface:** the exported types/functions other slices use, and what they promise.

**Design / flow:** how it works internally; the important decisions and why; the data
that flows in and out.

**Depends on:** which other slices/externals, and through what interface.

**Extending it:** what to do (and not do) when adding a feature here.
```

## 4. Code quality gate (must always pass)

Run automatically by `.githooks/pre-commit` on staged `.go` files; CI runs the same.
Enable hooks once: `git config core.hooksPath .githooks`.

1. **`gofmt`** — formatting is non-negotiable; the hook fails on any diff.
2. **`go vet ./...`** — zero findings.
3. **`golangci-lint run`** — zero findings (config in `.golangci.yml`; install per README).
4. **`go build ./...`** — must compile.
5. **`go test ./...`** — must pass. New behavior ships with a test.
6. **ABOUTME header** — present on every staged `.go` file.

Never bypass with `git commit --no-verify`.

## 5. Errors, secrets, logging

- Wrap errors with context and `%w`; never discard with `_` unless provably safe (comment why).
- Distinguish "no data" from "error" — don't collapse them into one return.
- Never log tokens/secrets/file contents that could be sensitive. Never commit `.env`.
- Structured logging; include the `job_id` / `machine_id` on every job-related log line.

## 6. Testing

- Table-driven tests for pure logic (queue transitions, header checks, config parsing).
- The executor and Supabase client are behind interfaces so they can be faked in tests.
- A feature isn't done until its `context.md` and tests are updated.

## 7. Workflow — small steps, commit each one

Build in small, verifiable steps. **Each step is its own commit** and must leave the
repo green:

1. Write/adjust the test for the step's behavior first.
2. Implement until the test passes.
3. Update the docs the step touches — the slice's `context.md`, `README.md`, and the
   flow/connection docs in `docs/` (see §8).
4. Stage everything and commit through the pre-commit gate (it must pass — never
   `--no-verify`). One focused change per commit, with a clear message.
5. Push.

Definition of done for a step: **tests pass + gate passes + docs updated + committed +
pushed.** Don't batch many unrelated changes into one commit.

## 8. Documentation — over-document the flow

Beacon is a moving system across cloud + laptop; the docs must let anyone trace a
command end-to-end without reading all the code. Keep these current **in the same commit
as the code they describe**:

- **`README.md`** — what it is, how to run each binary, how to set up the connection
  (Supabase project, env vars, tokens), and current status.
- **`docs/ARCHITECTURE.md`** — the living as-built doc: the end-to-end flow (you →
  Wingman → MCP → Supabase → agent → back), the job lifecycle/state machine, the
  **connection model** (what connects to what, in which direction, with which auth),
  data model, and the security model. Diagrams (ASCII/Mermaid) where they help.
- **`docs/CONNECTIONS.md`** — concrete connection/setup details: Supabase URL/keys,
  realtime channels, the agent↔Supabase and MCP↔Supabase wiring, token issuance and
  rotation, ports/firewall notes (none inbound — agent is outbound-only).
- **Per-folder `context.md`** — the design/flow of that slice (see §3).
- The **design spec** in `docs/superpowers/specs/` stays as the point-in-time design;
  `ARCHITECTURE.md` tracks how it actually got built and any deviations.

If behavior changes and a doc isn't updated in the same commit, the step isn't done.
