# Beacon Go Style Rules

Distilled from authoritative sources and wired into the pre-commit gate where a linter
can enforce them. Rules tagged **[gate]** are enforced automatically by
`golangci-lint` (`.golangci.yml`) and block the commit; **[review]** rules are checked
by humans/agents during review.

Sources: [Effective Go](https://go.dev/doc/effective_go),
[Google Go Style](https://google.github.io/styleguide/go/),
[Google Go best practices](https://google.github.io/styleguide/go/best-practices.html),
[Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).

## Formatting & imports
- **[gate]** All code is `gofumpt`-formatted (stricter superset of `gofmt`) with
  `goimports`-grouped imports. Formatters run in the gate; unformatted code is rejected.
- **[review]** Soft line limit ~99 chars; wrap for readability, not dogma.

## Naming
- **[gate]** No shadowing predeclared identifiers (`error`, `string`, `len`, …) — `predeclared`.
- **[gate]** Doc-comment + naming conventions via `revive` (`var-naming`, `exported`,
  `receiver-naming`, `package-comments`).
- **[review]** Packages: short, lowercase, singular, no underscores; never `util`/`common`/`helper`.
- **[review]** No `Get` prefix on value-returning funcs; verbs for actions, nouns for values.
- **[review]** Don't repeat the receiver/parameter type in a method name.
- **[review]** Variable name length scales with scope (short in tight scope, descriptive when wide).

## Documentation
- **[gate]** Every exported symbol has a doc comment starting with its name; comments end
  with a period — `revive` (`exported`) + `godot`.
- **[review]** Present tense, third person; document sentinel errors and non-obvious behavior.

## Error handling
- **[gate]** Wrap with `%w` and compare via `errors.Is`/`errors.As` (never `==` on wrapped
  errors, never type assertion on errors) — `errorlint`.
- **[gate]** All returned errors are checked — `errcheck`.
- **[review]** Error strings: lowercase, no trailing punctuation. `%w` at end of the message
  (at the start only to highlight a sentinel category).
- **[review]** Sentinels named `ErrXxx`; custom error types named `XxxError`.
- **[review]** Handle an error once: match/log it, OR wrap-and-return it — not both. Don't log
  an error you also return (let the caller decide).
- **[review]** Never panic for control flow; panic only on truly unrecoverable state.

## Context
- **[gate]** Never store `context.Context` in a struct — `containedctx`.
- **[gate]** Outbound HTTP/requests carry a context — `noctx`.
- **[review]** `context.Context` is the first parameter (`ctx ctx.Context`); never put it in
  an options struct.

## Interfaces & types
- **[review]** Accept interfaces, return concrete types.
- **[review]** Verify compliance at compile time: `var _ Iface = (*Impl)(nil)`.
- **[review]** Don't embed types (or `sync.Mutex`) in public structs — use named fields.
- **[gate]** Struct fields that are (un)marshaled must have explicit tags — `musttag`.

## Nil, zero values, allocation
- **[review]** Nil slices are valid; test emptiness with `len(s) == 0`.
- **[review]** Zero-value `sync.Mutex` is ready; use `var mu sync.Mutex`, not `new(...)`.
- **[gate]** No unnecessary type conversions — `unconvert`.
- **[gate]** No unused function parameters — `unparam`.
- **[review]** Pre-size maps/slices with `make(T, n)` when the size is known.

## Control flow
- **[gate]** Avoid deep nesting — `nestif`, `gocritic`.
- **[gate]** No naked returns in non-trivial functions — `nakedret`.
- **[review]** Handle errors/special cases first and return early; avoid `else` after a return.

## Concurrency
- **[gate]** Leak/lint checks via `gosec` + `govet` where detectable.
- **[review]** Every goroutine has a clear stop signal and is awaited (`WaitGroup` /
  `chan struct{}`); never spawn goroutines in `init()`.
- **[review]** Channel capacity is 0 or 1 unless strongly justified.
- **[review]** Add `go.uber.org/goleak` leak checks in package tests for code that spawns goroutines.

## Initialization & globals
- **[review]** Avoid `init()`; prefer constructors/`main()`. No mutable package-level globals —
  inject dependencies.
- **[review]** Enums start at 1 (`iota + 1`) so the zero value is an invalid sentinel.

## Security (remote-control tool — `gosec` is **[gate]**)
- **[gate]** `gosec` runs on every commit (command injection, path traversal, weak crypto, etc.).
- **[review]** Never interpolate untrusted input into a shell string without explicit, reviewed
  handling; never log tokens/secrets; validate and bound all paths the agent touches.

## Testing
- **[review]** Table-driven tests with named struct fields; `t.Helper()` in helpers.
- **[review]** `t.Fatal` only in setup/helpers; `t.Error`+continue per table entry. Never
  `t.Fatal`/`FailNow` from a non-test goroutine.
- **[review]** Keep assertions in the test body; avoid heavy assertion-helper indirection.

## Tooling
- **[gate]** `nolintlint` requires every `//nolint` to be specific and explained — no blanket suppressions.
