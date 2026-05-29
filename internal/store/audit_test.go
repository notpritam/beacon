// ABOUTME: DB-backed tests for appending and listing audit entries.
// ABOUTME: Audit is append-only; ListAudit returns newest first.
package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestAppendAndListAudit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	mid := seedMachine(t, s)
	j, err := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{}`), 0, nil, "")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if err := s.AppendAudit(ctx, j.ID, mid, "created", json.RawMessage(`{"by":"wingman"}`)); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := s.AppendAudit(ctx, j.ID, mid, "done", nil); err != nil {
		t.Fatalf("append 2: %v", err)
	}

	entries, err := s.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Event != "done" {
		t.Errorf("newest event = %q, want done", entries[0].Event)
	}
}

func TestAppendAuditWithNullIDs(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.AppendAudit(ctx, "", "", "system-event", nil); err != nil {
		t.Fatalf("append: %v", err)
	}
	entries, err := s.ListAudit(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].JobID != "" || entries[0].MachineID != "" {
		t.Errorf("expected empty ids, got job=%q machine=%q", entries[0].JobID, entries[0].MachineID)
	}
	if entries[0].Event != "system-event" {
		t.Errorf("event = %q", entries[0].Event)
	}
}

func TestListAuditRespectsLimit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	for _, ev := range []string{"a", "b", "c"} {
		if err := s.AppendAudit(ctx, "", "", ev, nil); err != nil {
			t.Fatalf("append %s: %v", ev, err)
		}
	}
	entries, err := s.ListAudit(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1", len(entries))
	}
	if entries[0].Event != "c" {
		t.Errorf("newest = %q, want c", entries[0].Event)
	}
}
