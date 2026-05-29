# localaudit — context

**Purpose:** Append-only local mirror of job audit events, on disk, independent of the DB.

**Public surface:** `Entry` (at, job_id, machine_id, event, detail), `Logger`, `New(path)`,
`(*Logger).Append(entry) error`.

**Design / flow:** `Append` JSON-marshals the entry and writes it as one line to the file
(`O_APPEND|O_CREATE`), under a mutex so concurrent callers are safe. A missing timestamp is
stamped with `time.Now()`. The mirror exists so the audit trail survives even if the cloud
`audit_log` row is tampered with or unreachable.

**Depends on:** stdlib (`os`, `encoding/json`, `sync`, `time`).

**Extending it:** keep it append-only and store-independent; do not add read/query logic
here (read the file directly or ship a separate reader).
