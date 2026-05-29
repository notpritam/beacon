// ABOUTME: beaconagent is the laptop daemon: it registers the machine and drains its queue.
// ABOUTME: Wires config -> store -> executor -> localaudit -> agent and runs under a signal context.

// Command beaconagent runs the Beacon laptop agent.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/notpritam/beacon/internal/agent"
	"github.com/notpritam/beacon/internal/config"
	"github.com/notpritam/beacon/internal/dashboard"
	"github.com/notpritam/beacon/internal/executor"
	"github.com/notpritam/beacon/internal/localaudit"
	"github.com/notpritam/beacon/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name := cfg.MachineName
	if name == "" {
		h, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("resolve hostname: %w", err)
		}
		name = h
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}
	stateDir := filepath.Join(home, ".beacon")
	if err := os.MkdirAll(stateDir, 0o755); err != nil { //nolint:gosec // agent-owned state dir
		return fmt.Errorf("create state dir: %w", err)
	}
	auditPath := filepath.Join(stateDir, "audit.log")
	sentinel := filepath.Join(stateDir, "killswitch")

	exec := executor.New(executor.DefaultConfig())
	ag := agent.New(st, exec, localaudit.New(auditPath), sentinel, agent.Options{})
	if err := ag.Register(ctx, name, runtime.GOOS, cfg.MachineToken); err != nil {
		return err
	}

	if cfg.DashboardToken != "" {
		startDashboard(ctx, cfg, exec, st, ag.MachineID())
	}

	fmt.Printf("beacon-agent running as %q (audit: %s, kill-switch sentinel: %s)\n", name, auditPath, sentinel)
	if err := ag.Run(ctx); err != nil && ctx.Err() == nil {
		return err
	}
	fmt.Println("beacon-agent stopped")
	return nil
}

// startDashboard launches the live-view dashboard. The screen is only captured
// while a viewer is connected (the MJPEG stream runs per-request), so it is idle
// when nobody is watching. It is gated by BEACON_DASHBOARD_TOKEN.
func startDashboard(ctx context.Context, cfg config.Config, exec *executor.Executor, st *store.Store, machineID string) {
	jobsFn := func(ctx context.Context) (any, error) {
		js, err := st.RecentJobs(ctx, machineID, 20)
		if err != nil {
			return nil, err
		}
		type view struct {
			ID        string    `json:"id"`
			Type      string    `json:"type"`
			Status    string    `json:"status"`
			CreatedAt time.Time `json:"created_at"`
			Result    string    `json:"result"`
		}
		out := make([]view, 0, len(js))
		for _, j := range js {
			out = append(out, view{
				ID: j.ID, Type: string(j.Type), Status: string(j.Status),
				CreatedAt: j.CreatedAt, Result: summarizeResult(j),
			})
		}
		return out, nil
	}

	dash := dashboard.New(cfg.DashboardToken, exec.CaptureScreenshot, jobsFn, 3)
	srv := &http.Server{Addr: cfg.DashboardAddr, Handler: dash.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "dashboard:", err)
		}
	}()
	fmt.Printf("beacon dashboard on %s — open /?token=<BEACON_DASHBOARD_TOKEN>; the screen streams only while a viewer is connected\n", cfg.DashboardAddr)
}

// summarizeResult returns a short, feed-friendly string for a job's result,
// avoiding huge payloads like screenshot base64.
func summarizeResult(j store.Job) string {
	if len(j.Result) == 0 {
		return ""
	}
	if j.Type == store.JobScreenshot {
		return "screenshot captured"
	}
	s := string(j.Result)
	const maxLen = 200
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}
