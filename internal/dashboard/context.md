# dashboard — context

**Purpose:** Serve a browser live view of the machine — an MJPEG screen stream plus a
recent-jobs feed — so a human can *watch* what Wingman is doing. It does not replace the
`screenshot` job (which is how the agent itself sees the screen); it is observation only.

**Public surface:** `New(token string, capture CaptureFunc, jobs JobsFunc, fps int) *Server`
and `(*Server).Handler() http.Handler`. The capture and jobs functions are injected by the
caller (`cmd/agent`), so this package has no dependency on the executor or store.

**Routes (all token-gated):**
| Path | Returns |
|---|---|
| `/` | HTML page: live `<img>` + jobs feed (polls `/jobs` every 2s) |
| `/live` | `multipart/x-mixed-replace` JPEG stream (native `<img>` live view) |
| `/jobs` | JSON array of recent jobs |

**Design / flow:**
- **On-demand capture.** Frames are produced *inside* the `/live` request loop, so the
  screen is only captured while a viewer is connected. When the browser tab closes, the
  request context is cancelled and capture stops. Nothing is captured when nobody watches.
- **Auth.** Every route is wrapped by `auth`, a constant-time compare of `?token=` against
  the configured token → 401 on mismatch. The index page injects the *server's own* token
  (post-auth) into the HTML, never request-derived data, to avoid reflected XSS.
- **Framing.** `/live` writes `--beaconframe` boundaries with `Content-Type: image/jpeg`
  per frame and flushes after each; the loop sleeps `1s / fps` between frames.

**Depends on:** stdlib only (`net/http`, `crypto/subtle`, `encoding/json`). The capture
function is `executor.CaptureScreenshot`; the jobs function wraps `store.RecentJobs`.

**Wiring:** the agent starts the dashboard only when `BEACON_DASHBOARD_TOKEN` is set
(see `cmd/agent/main.go:startDashboard`). It is exposed to the operator over a tunnel
(e.g. a Cloudflare quick tunnel) separate from the MCP tunnel.

**Extending it:** keep capture/jobs injected — do not import the executor or store here.
New panels should follow the same token-gated, on-demand pattern (no always-on work).
