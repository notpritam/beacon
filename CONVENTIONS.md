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
