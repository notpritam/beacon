# cmd/mcp — Beacon MCP Server

## Purpose

`cmd/mcp` is the cloud-side binary that Wingman (the Emergent cloud agent) talks to.
It exposes Beacon's 7 machine-control tools over the MCP **streamable HTTP** transport,
protected by bearer-token authentication.

## How to run

```bash
BEACON_DATABASE_URL=<postgres-url> BEACON_WINGMAN_TOKEN=<secret> go run ./cmd/mcp
```

The server prints `beacon-mcp listening on <addr>/mcp` on start.

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `BEACON_DATABASE_URL` | yes | — | Postgres connection string (Supabase or local). |
| `BEACON_WINGMAN_TOKEN` | yes | — | Bearer token Wingman must present. Server refuses to start without it. |
| `BEACON_MCP_ADDR` | no | `:8080` | TCP listen address (e.g. `:8765`). |

## Exposed tools (all via `/mcp`)

| Tool | Description |
|---|---|
| `list_machines` | List all registered machines with online/offline status. |
| `machine_status` | Get a single machine's status, last_seen, and kill-switch state. |
| `run_command` | Run a shell command; long-polls if online, returns job_id if offline. |
| `read_file` | Read a file on a machine. |
| `write_file` | Write content to a file on a machine. |
| `list_dir` | List a directory on a machine. |
| `get_job` | Fetch the current status and result of a job by ID. |

## Authentication

Every request to `/mcp` must carry `Authorization: Bearer <BEACON_WINGMAN_TOKEN>`.
The check is constant-time (`crypto/subtle`). Requests without a valid token receive
`401 Unauthorized` before reaching the MCP handler. The server **fails closed**:
if `BEACON_WINGMAN_TOKEN` is unset it exits immediately at startup.

## Transport

The MCP protocol is served as **streamable HTTP** per the
[MCP 2025-03-26 spec](https://modelcontextprotocol.io/2025/03/26/streamable-http-transport.html)
via `github.com/modelcontextprotocol/go-sdk/mcp.NewStreamableHTTPHandler`. A single
`*mcp.Server` instance is shared across sessions.

The `http.Server` sets `ReadHeaderTimeout: 10s` to avoid slow-loris exposure.

## Dependencies

- `internal/config` — loads env vars.
- `internal/store` — Postgres connection (`store.New`, `store.Close`).
- `internal/mcptools` — implements the tool logic over the store.
- `github.com/modelcontextprotocol/go-sdk/mcp` — MCP protocol and streamable HTTP handler.

## Extending

To add a new tool: add an input struct with `json`/`jsonschema` tags in `tools.go`,
then call `mcp.AddTool` in `registerTools`. The SDK infers the JSON Schema from the
struct. The output type must be a struct (not a slice) so the SDK can infer an
"object" schema; wrap slices in a container struct as shown with `machineList`.
