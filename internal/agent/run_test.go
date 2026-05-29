// ABOUTME: Integration tests for RunOnce/Run against a real store + real executor.
// ABOUTME: Each test uses a unique machine name (t.Name()) for isolation.
package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func registeredAgent(t *testing.T, st *store.Store) (*Agent, string) {
	t.Helper()
	a := newTestAgent(t, st)
	if err := a.Register(context.Background(), t.Name(), "darwin", "tok"); err != nil {
		t.Fatalf("register: %v", err)
	}
	return a, a.machineID
}

func TestRunOnceExecutesShellJob(t *testing.T) {
	st := newAgentStore(t)
	a, mid := registeredAgent(t, st)
	ctx := context.Background()
	if _, err := st.EnqueueJob(ctx, mid, store.JobShell, json.RawMessage(`{"cmd":"echo hi"}`), 0, nil, "test"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	did, err := a.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !did {
		t.Fatal("expected RunOnce to do work")
	}
	claimed, _ := st.ClaimNextJob(ctx, mid)
	if claimed != nil {
		t.Error("queue should be empty after processing")
	}
}

func TestRunOnceEmptyQueue(t *testing.T) {
	st := newAgentStore(t)
	a, _ := registeredAgent(t, st)
	did, err := a.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if did {
		t.Error("expected no work on empty queue")
	}
}

func TestRunOnceFailsUnsupportedJob(t *testing.T) {
	st := newAgentStore(t)
	a, mid := registeredAgent(t, st)
	ctx := context.Background()
	j, err := st.EnqueueJob(ctx, mid, store.JobScreenshot, json.RawMessage(`{}`), 0, nil, "test")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := a.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}
	got, err := st.GetJob(ctx, j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != store.JobFailed {
		t.Errorf("status = %q, want failed", got.Status)
	}
}

func TestRunOnceSkipsWhenKillSwitchTripped(t *testing.T) {
	st := newAgentStore(t)
	a, mid := registeredAgent(t, st)
	ctx := context.Background()
	if err := st.SetKillSwitch(ctx, mid, true); err != nil {
		t.Fatalf("set kill: %v", err)
	}
	if _, err := st.EnqueueJob(ctx, mid, store.JobShell, json.RawMessage(`{"cmd":"echo hi"}`), 0, nil, "test"); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	did, err := a.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if did {
		t.Error("expected no work while kill switch is tripped")
	}
	claimed, _ := st.ClaimNextJob(ctx, mid)
	if claimed == nil {
		t.Error("job should still be queued when kill switch is tripped")
	}
}
