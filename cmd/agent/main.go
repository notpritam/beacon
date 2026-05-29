// ABOUTME: beaconagent is the laptop daemon: it registers the machine and drains its queue.
// ABOUTME: Wires config -> store -> executor -> localaudit -> agent and runs under a signal context.

// Command beaconagent runs the Beacon laptop agent.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/notpritam/beacon/internal/agent"
	"github.com/notpritam/beacon/internal/config"
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

	ag := agent.New(st, executor.New(executor.DefaultConfig()), localaudit.New(auditPath), sentinel, agent.Options{})
	if err := ag.Register(ctx, name, runtime.GOOS, cfg.MachineToken); err != nil {
		return err
	}

	fmt.Printf("beacon-agent running as %q (audit: %s, kill-switch sentinel: %s)\n", name, auditPath, sentinel)
	if err := ag.Run(ctx); err != nil && ctx.Err() == nil {
		return err
	}
	fmt.Println("beacon-agent stopped")
	return nil
}
