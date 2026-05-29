// ABOUTME: DB-backed tests for the read-only MCP tools (list/status/get_job).
// ABOUTME: Each test uses a unique machine name; skips without TEST_DATABASE_URL.
package mcptools

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func newToolsStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-backed test")
	}
	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestMachineStatusOnlineOffline(t *testing.T) {
	st := newToolsStore(t)
	tools := New(st, Options{})
	ctx := context.Background()
	m, err := st.RegisterMachine(ctx, t.Name(), "darwin", "h")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	info, err := tools.MachineStatus(ctx, t.Name())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if info.Online {
		t.Error("freshly registered machine (no heartbeat) should be offline")
	}
	if err := st.Heartbeat(ctx, m.ID); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	info, err = tools.MachineStatus(ctx, t.Name())
	if err != nil {
		t.Fatalf("status 2: %v", err)
	}
	if !info.Online {
		t.Error("machine should be online right after heartbeat")
	}
}

func TestMachineStatusUnknown(t *testing.T) {
	st := newToolsStore(t)
	tools := New(st, Options{})
	if _, err := tools.MachineStatus(context.Background(), "no-such-machine"); err == nil {
		t.Error("expected error for unknown machine")
	}
}

func TestGetJob(t *testing.T) {
	st := newToolsStore(t)
	tools := New(st, Options{})
	ctx := context.Background()
	m, err := st.RegisterMachine(ctx, t.Name(), "darwin", "h")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	j, err := st.EnqueueJob(ctx, m.ID, store.JobShell, json.RawMessage(`{"cmd":"echo hi"}`), 0, nil, "test")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	out, err := tools.GetJob(ctx, j.ID)
	if err != nil {
		t.Fatalf("get_job: %v", err)
	}
	if out.JobID != j.ID || out.Status != string(store.JobQueued) {
		t.Errorf("unexpected outcome: %+v", out)
	}
}

func TestListMachines(t *testing.T) {
	st := newToolsStore(t)
	tools := New(st, Options{})
	ctx := context.Background()
	if _, err := st.RegisterMachine(ctx, t.Name(), "darwin", "h"); err != nil {
		t.Fatalf("register: %v", err)
	}
	ms, err := tools.ListMachines(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, m := range ms {
		if m.Name == t.Name() {
			found = true
		}
	}
	if !found {
		t.Error("registered machine not in list")
	}
}
