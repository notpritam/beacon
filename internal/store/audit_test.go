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
	j, _ := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{}`), 0, nil, "")

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
