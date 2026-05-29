// ABOUTME: A small live-view web dashboard: an MJPEG screen stream + a recent-jobs feed.
// ABOUTME: Token-gated via ?token=. Served by the agent; capture/jobs funcs are injected.

// Package dashboard serves a browser live view of the machine (screen + job feed).
package dashboard

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CaptureFunc returns a single JPEG frame of the screen.
type CaptureFunc func(ctx context.Context) ([]byte, error)

// JobsFunc returns a JSON-serializable view of recent jobs.
type JobsFunc func(ctx context.Context) (any, error)

// Server serves the live dashboard.
type Server struct {
	token   string
	capture CaptureFunc
	jobs    JobsFunc
	fps     int
}

// New returns a dashboard Server. fps bounds the live stream frame rate (default 2).
func New(token string, capture CaptureFunc, jobs JobsFunc, fps int) *Server {
	if fps <= 0 {
		fps = 2
	}
	return &Server{token: token, capture: capture, jobs: jobs, fps: fps}
}

// Handler returns the dashboard's HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.auth(s.handleIndex))
	mux.HandleFunc("/live", s.auth(s.handleLive))
	mux.HandleFunc("/jobs", s.auth(s.handleJobs))
	return mux
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := r.URL.Query().Get("token")
		if subtle.ConstantTimeCompare([]byte(tok), []byte(s.token)) != 1 {
			http.Error(w, "unauthorized — append ?token=<BEACON_WINGMAN_TOKEN>", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	// Inject the server's own token (auth already proved the request's token equals
	// it), so no request-derived data is reflected into the page.
	html := strings.ReplaceAll(indexHTML, "{{TOKEN}}", s.token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// handleLive streams JPEG frames as multipart/x-mixed-replace (native <img> live view).
func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	const boundary = "beaconframe"
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+boundary)
	w.Header().Set("Cache-Control", "no-store")
	interval := time.Second / time.Duration(s.fps)
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		frame, err := s.capture(ctx)
		if err != nil {
			time.Sleep(interval)
			continue
		}
		if _, err := fmt.Fprintf(w, "--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", boundary, len(frame)); err != nil {
			return
		}
		if _, err := w.Write(frame); err != nil {
			return
		}
		if _, err := w.Write([]byte("\r\n")); err != nil {
			return
		}
		flusher.Flush()
		time.Sleep(interval)
	}
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	data, err := s.jobs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

const indexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>Beacon — my-mac</title>
<style>
  body{margin:0;background:#0b0d10;color:#cdd3da;font:13px/1.4 -apple-system,system-ui,sans-serif;display:flex;height:100vh}
  #screen{flex:3;display:flex;align-items:center;justify-content:center;background:#000;overflow:hidden}
  #screen img{max-width:100%;max-height:100%;object-fit:contain}
  #side{flex:1;min-width:280px;max-width:420px;border-left:1px solid #222;display:flex;flex-direction:column}
  h2{margin:0;padding:10px 12px;font-size:12px;letter-spacing:.08em;text-transform:uppercase;color:#7d8590;border-bottom:1px solid #222}
  #jobs{overflow:auto;padding:6px 0}
  .job{padding:7px 12px;border-bottom:1px solid #16191d}
  .row{display:flex;justify-content:space-between;gap:8px}
  .type{font-weight:600;color:#e6edf3}
  .st-done{color:#3fb950}.st-failed{color:#f85149}.st-queued,.st-running,.st-claimed{color:#d29922}
  .res{color:#7d8590;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;margin-top:2px}
  .t{color:#586069;font-size:11px}
</style></head>
<body>
  <div id="screen"><img src="/live?token={{TOKEN}}" alt="live screen"></div>
  <div id="side"><h2>Recent jobs</h2><div id="jobs">loading…</div></div>
<script>
const tok="{{TOKEN}}";
async function refresh(){
  try{
    const r=await fetch("/jobs?token="+encodeURIComponent(tok));
    const jobs=await r.json();
    document.getElementById("jobs").innerHTML=(jobs||[]).map(j=>
      '<div class="job"><div class="row"><span class="type">'+j.type+'</span>'+
      '<span class="st-'+j.status+'">'+j.status+'</span></div>'+
      '<div class="res">'+(j.result||'')+'</div>'+
      '<div class="t">'+j.created_at+'</div></div>').join('');
  }catch(e){}
}
refresh();setInterval(refresh,2000);
</script>
</body></html>`
