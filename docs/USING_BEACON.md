# Using Beacon from an Agent

Beacon lets an agent run shell commands and file operations on a registered laptop, by
calling **MCP tools** over HTTP. This is the integration guide for any agent (Wingman or
other) that should drive a machine.

## Connect

Beacon's MCP server speaks **streamable HTTP** at the `/mcp` path, in **stateless** mode
with **JSON responses** — so a client can POST a JSON-RPC tool call directly (no session
handshake, no SSE parsing). Every request must carry the bearer token.

- **Endpoint:** `<MCP_URL>` (e.g. `https://your-host/mcp`)
- **Auth header:** `Authorization: Bearer <WINGMAN_TOKEN>`
- **Content-Type:** `application/json` · **Accept:** `application/json, text/event-stream`

Requests without a valid token get `401`. The token grants full command execution on the
target machine — treat it like a root password.

### Register in an MCP client

- **Claude / Claude Code** (`~/.claude/mcp.json` or project `.mcp.json`):
  ```json
  {
    "mcpServers": {
      "beacon": {
        "type": "http",
        "url": "<MCP_URL>",
        "headers": { "Authorization": "Bearer <WINGMAN_TOKEN>" }
      }
    }
  }
  ```
- **Wingman / other agents:** add an HTTP (streamable) MCP server with the same URL and
  `Authorization` header. The agent then sees the tools below natively.

## Targeting a machine

Tools that act on a laptop take a `machine` argument — the machine's registered **name**
(its `BEACON_MACHINE_NAME`, e.g. `my-mac`). Call `list_machines` to discover names and
whether each is **online** (heartbeating). If a machine is offline, action tools return a
`job_id` with `status:"queued"`; the job runs when the laptop reconnects, and you can poll
`get_job`.

## Tools

| Tool | Arguments | Returns |
|---|---|---|
| `list_machines` | — | `{ machines: [{name, os, online, last_seen, kill_switch}] }` |
| `machine_status` | `machine` | `{name, os, online, last_seen, kill_switch}` |
| `run_command` | `machine`, `cmd`, `cwd?`, `timeout_secs?` | `JobOutcome` (result `{stdout, stderr, exit_code}`) |
| `read_file` | `machine`, `path` | `JobOutcome` (result `{content}`) |
| `write_file` | `machine`, `path`, `content` | `JobOutcome` (result `{bytes_written}`) |
| `list_dir` | `machine`, `path` | `JobOutcome` (result `{entries:[{name,is_dir,size}]}`) |
| `get_job` | `job_id` | `JobOutcome` |

**`JobOutcome`** = `{ job_id, status, machine_online, result }`. `status` is one of
`queued` / `claimed` / `running` / `done` / `failed` / `expired`. When the machine is
online, action tools wait for the job to finish and return the `result`; when offline they
return immediately with `status:"queued"` and an empty `result` — poll `get_job` later.

## Example — run a command

Request (raw JSON-RPC; most agents just "call the `run_command` tool"):
```bash
curl -s -X POST "<MCP_URL>" \
  -H "Authorization: Bearer <WINGMAN_TOKEN>" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{
    "name":"run_command",
    "arguments":{"machine":"my-mac","cmd":"echo hi && whoami","timeout_secs":30}}}'
```
Response (`result.structuredContent`):
```json
{
  "job_id": "…",
  "status": "done",
  "machine_online": true,
  "result": { "stdout": "hi\nyou\n", "stderr": "", "exit_code": 0 }
}
```

## Guidance for the agent

- Use `list_machines` first to find the right `machine` and confirm it's `online`.
- A non-zero `exit_code` is a **successful run** with a failing command — read `stderr`.
- `status:"failed"` means the job could not execute (bad path, unsupported type); `result`
  carries an `error`.
- For long or offline work, expect `status:"queued"`/`"running"` and poll `get_job`.
- `screenshot` and background/long-running jobs are **not available yet**.

## Safety

The operator can stop the agent at any time via the kill switch (a cloud flag or a local
sentinel file `~/.beacon/killswitch`). While engaged, the agent stops claiming new jobs.
Every action is recorded in an append-only audit log (cloud + local `~/.beacon/audit.log`).
