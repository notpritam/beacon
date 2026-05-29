# config — context

**Purpose:** Load and validate Beacon settings from environment variables.

**Public surface:** `Config` struct and `Load() (Config, error)`. `Load` returns an
error if a required variable (currently `BEACON_DATABASE_URL`) is missing.

**Design / flow:** Pure function over `os.Getenv`; no I/O beyond env reads. Each binary
calls `Load` once at startup and passes `Config` down explicitly (no globals).

**Depends on:** stdlib only.

**Extending it:** add new env keys as fields here and validate them in `Load`; never read
env vars elsewhere in the codebase.
