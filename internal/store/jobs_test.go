// ABOUTME: DB-backed tests for enqueue, atomic claim, lifecycle transitions, and TTL expiry.
// ABOUTME: The claim test asserts a job is handed out at most once.
package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func seedMachine(t *testing.T, s *Store) string {
	t.Helper()
	m, err := s.RegisterMachine(context.Background(), "mac-1", "darwin", "h")
	if err != nil {
		t.Fatalf("seed machine: %v", err)
	}
	return m.ID
}

func TestEnqueueAndGetJob(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	mid := seedMachine(t, s)

	j, err := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{"cmd":"echo hi"}`), 0, nil, "wingman")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if j.Status != JobQueued {
		t.Errorf("status = %q, want queued", j.Status)
	}

	got, err := s.GetJob(ctx, j.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Type != JobShell || got.CreatedBy != "wingman" {
		t.Errorf("unexpected job: %+v", got)
	}
}

func TestEnqueueRejectsInvalidType(t *testing.T) {
	s := newTestStore(t)
	mid := seedMachine(t, s)
	if _, err := s.EnqueueJob(context.Background(), mid, JobType("bogus"), json.RawMessage(`{}`), 0, nil, ""); err == nil {
		t.Error("expected error for invalid job type")
	}
}

func TestClaimNextJobIsAtomic(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	mid := seedMachine(t, s)

	if _, err := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{}`), 0, nil, ""); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	first, err := s.ClaimNextJob(ctx, mid)
	if err != nil {
		t.Fatalf("claim 1: %v", err)
	}
	if first == nil {
		t.Fatal("expected a job on first claim")
	}
	if first.Status != JobClaimed {
		t.Errorf("claimed status = %q, want claimed", first.Status)
	}

	second, err := s.ClaimNextJob(ctx, mid)
	if err != nil {
		t.Fatalf("claim 2: %v", err)
	}
	if second != nil {
		t.Errorf("second claim should return nil, got %s", second.ID)
	}
}

func TestClaimRespectsPriority(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	mid := seedMachine(t, s)

	if _, err := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{"n":"low"}`), 0, nil, ""); err != nil {
		t.Fatalf("enqueue low: %v", err)
	}
	if _, err := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{"n":"high"}`), 10, nil, ""); err != nil {
		t.Fatalf("enqueue high: %v", err)
	}

	claimed, err := s.ClaimNextJob(ctx, mid)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected a job")
	}
	var got struct {
		N string `json:"n"`
	}
	if err := json.Unmarshal(claimed.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got.N != "high" {
		t.Errorf("expected high-priority job first, got %q", got.N)
	}
}

func TestCompleteAndFailJob(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	mid := seedMachine(t, s)

	j, _ := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{}`), 0, nil, "")
	if err := s.StartJob(ctx, j.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := s.CompleteJob(ctx, j.ID, json.RawMessage(`{"exit":0}`)); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, _ := s.GetJob(ctx, j.ID)
	if got.Status != JobDone || got.FinishedAt == nil {
		t.Errorf("expected done+finished, got %+v", got)
	}

	j2, _ := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{}`), 0, nil, "")
	if err := s.FailJob(ctx, j2.ID, json.RawMessage(`{"error":"boom"}`)); err != nil {
		t.Fatalf("fail: %v", err)
	}
	got2, _ := s.GetJob(ctx, j2.ID)
	if got2.Status != JobFailed {
		t.Errorf("expected failed, got %q", got2.Status)
	}
}

func TestExpireDueJobs(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	mid := seedMachine(t, s)

	past := time.Now().Add(-time.Hour)
	if _, err := s.EnqueueJob(ctx, mid, JobShell, json.RawMessage(`{}`), 0, &past, ""); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	n, err := s.ExpireDueJobs(ctx)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if n != 1 {
		t.Errorf("expired = %d, want 1", n)
	}
	claimed, _ := s.ClaimNextJob(ctx, mid)
	if claimed != nil {
		t.Error("expired job should not be claimable")
	}
}
