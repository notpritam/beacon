// ABOUTME: DB-backed tests for machine registration, heartbeat, and kill switch.
// ABOUTME: Requires TEST_DATABASE_URL (otherwise skipped via newTestStore).
package store

import (
	"context"
	"errors"
	"testing"
)

func TestRegisterAndGetMachine(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	m, err := s.RegisterMachine(ctx, "mac-1", "darwin", "hash123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if m.ID == "" {
		t.Fatal("expected non-empty id")
	}

	got, err := s.MachineByName(ctx, "mac-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != m.ID || got.OS != "darwin" || got.KillSwitch {
		t.Errorf("unexpected machine: %+v", got)
	}
}

func TestRegisterMachineIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	m1, err := s.RegisterMachine(ctx, "mac-1", "darwin", "hashA")
	if err != nil {
		t.Fatalf("register 1: %v", err)
	}
	m2, err := s.RegisterMachine(ctx, "mac-1", "darwin", "hashB")
	if err != nil {
		t.Fatalf("register 2: %v", err)
	}
	if m1.ID != m2.ID {
		t.Errorf("re-register changed id: %s vs %s", m1.ID, m2.ID)
	}
	if m2.TokenHash != "hashB" {
		t.Errorf("token not updated: %q", m2.TokenHash)
	}
}

func TestHeartbeatAndKillSwitch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	m, err := s.RegisterMachine(ctx, "mac-1", "darwin", "h")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := s.Heartbeat(ctx, m.ID); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	got, err := s.MachineByName(ctx, "mac-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.LastSeen == nil {
		t.Fatal("last_seen should be set after heartbeat")
	}

	if err := s.SetKillSwitch(ctx, m.ID, true); err != nil {
		t.Fatalf("set kill: %v", err)
	}
	got, err = s.MachineByName(ctx, "mac-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.KillSwitch {
		t.Error("kill switch should be true")
	}

	if err := s.SetKillSwitch(ctx, m.ID, false); err != nil {
		t.Fatalf("clear kill: %v", err)
	}
	got, err = s.MachineByName(ctx, "mac-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.KillSwitch {
		t.Error("kill switch should be false after toggle off")
	}
}

func TestMachineByNameNotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.MachineByName(context.Background(), "ghost"); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestHeartbeatUnknownMachine(t *testing.T) {
	s := newTestStore(t)
	const ghostID = "00000000-0000-0000-0000-000000000000"
	if err := s.Heartbeat(context.Background(), ghostID); !errors.Is(err, ErrNotFound) {
		t.Errorf("heartbeat: want ErrNotFound, got %v", err)
	}
	if err := s.SetKillSwitch(context.Background(), ghostID, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("set kill: want ErrNotFound, got %v", err)
	}
}

func TestListMachines(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if _, err := s.RegisterMachine(ctx, "mac-1", "darwin", "h1"); err != nil {
		t.Fatalf("reg1: %v", err)
	}
	if _, err := s.RegisterMachine(ctx, "mac-2", "linux", "h2"); err != nil {
		t.Fatalf("reg2: %v", err)
	}
	ms, err := s.ListMachines(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ms) != 2 {
		t.Errorf("len = %d, want 2", len(ms))
	}
}

func TestMachineKillSwitch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	m, err := s.RegisterMachine(ctx, "mac-1", "darwin", "h")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	on, err := s.MachineKillSwitch(ctx, m.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if on {
		t.Error("kill switch should default to false")
	}

	if err := s.SetKillSwitch(ctx, m.ID, true); err != nil {
		t.Fatalf("set: %v", err)
	}
	on, err = s.MachineKillSwitch(ctx, m.ID)
	if err != nil {
		t.Fatalf("read 2: %v", err)
	}
	if !on {
		t.Error("kill switch should be true after set")
	}

	if _, err := s.MachineKillSwitch(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown machine: want ErrNotFound, got %v", err)
	}
}
