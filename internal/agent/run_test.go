// ABOUTME: Integration tests for RunOnce/Run against a real store + real executor.
// ABOUTME: Each test uses a unique machine name (t.Name()) for isolation.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/notpritam/beacon/internal/executor"
	"github.com/notpritam/beacon/internal/localaudit"
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
	j, err := st.EnqueueJob(ctx, mid, store.JobShell, json.RawMessage(`{"cmd":"echo hi"}`), 0, nil, "test")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	did, err := a.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !did {
		t.Fatal("expected RunOnce to do work")
	}
	got, err := st.GetJob(ctx, j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != store.JobDone {
		t.Errorf("status = %q, want done", got.Status)
	}
	var res struct {
		Stdout string `json:"stdout"`
	}
	if err := json.Unmarshal(got.Result, &res); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !strings.Contains(res.Stdout, "hi") {
		t.Errorf("stdout = %q, want it to contain hi", res.Stdout)
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

type fakeStore struct {
	job      *store.Job
	startErr error
}

func (f *fakeStore) RegisterMachine(_ context.Context, name, _, _ string) (store.Machine, error) {
	return store.Machine{ID: "m-fake", Name: name}, nil
}
func (f *fakeStore) Heartbeat(_ context.Context, _ string) error                  { return nil }
func (f *fakeStore) ClaimNextJob(_ context.Context, _ string) (*store.Job, error) { return f.job, nil }
func (f *fakeStore) StartJob(_ context.Context, _ string) error                   { return f.startErr }
func (f *fakeStore) CompleteJob(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}
func (f *fakeStore) FailJob(_ context.Context, _ string, _ json.RawMessage) error { return nil }
func (f *fakeStore) AppendAudit(_ context.Context, _, _, _ string, _ json.RawMessage) error {
	return nil
}

func (f *fakeStore) MachineKillSwitch(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func TestRunOnceInfraErrorPropagates(t *testing.T) {
	fs := &fakeStore{
		job:      &store.Job{ID: "j1", Type: store.JobShell, Payload: json.RawMessage(`{"cmd":"echo hi"}`)},
		startErr: errors.New("db down"),
	}
	a := New(fs, executor.New(executor.DefaultConfig()), localaudit.New(t.TempDir()+"/a.log"), "", Options{})
	if err := a.Register(context.Background(), "m", "darwin", "tok"); err != nil {
		t.Fatalf("register: %v", err)
	}
	did, err := a.RunOnce(context.Background())
	if !did || err == nil {
		t.Errorf("expected (true, non-nil err) on infra failure, got (%v, %v)", did, err)
	}
}

func TestRunProcessesThenStops(t *testing.T) {
	st := newAgentStore(t)
	a, mid := registeredAgent(t, st)
	ctx := context.Background()
	j, err := st.EnqueueJob(ctx, mid, store.JobShell, json.RawMessage(`{"cmd":"echo hi"}`), 0, nil, "test")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := a.Run(runCtx); err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run should return the context error, got %v", err)
	}

	got, err := st.GetJob(ctx, j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != store.JobDone {
		t.Errorf("status = %q, want done", got.Status)
	}
	var res struct {
		Stdout string `json:"stdout"`
	}
	if err := json.Unmarshal(got.Result, &res); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if res.Stdout == "" {
		t.Error("expected stdout in result")
	}

	m, err := st.MachineByName(ctx, t.Name())
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if m.LastSeen == nil {
		t.Error("Run should have heartbeated (last_seen set)")
	}
}
