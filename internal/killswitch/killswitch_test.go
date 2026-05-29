// ABOUTME: Tests for the kill-switch Checker — local sentinel precedence and cloud flag.
// ABOUTME: Uses a fake CloudFlag and a temp sentinel; no database.
package killswitch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeCloud struct {
	on  bool
	err error
}

func (f fakeCloud) MachineKillSwitch(_ context.Context, _ string) (bool, error) {
	return f.on, f.err
}

func TestTrippedCloudFlag(t *testing.T) {
	c := New(fakeCloud{on: true}, "m1", "")
	tripped, err := c.Tripped(context.Background())
	if err != nil {
		t.Fatalf("tripped: %v", err)
	}
	if !tripped {
		t.Error("expected tripped when cloud flag is on")
	}
}

func TestNotTripped(t *testing.T) {
	c := New(fakeCloud{on: false}, "m1", "")
	tripped, err := c.Tripped(context.Background())
	if err != nil {
		t.Fatalf("tripped: %v", err)
	}
	if tripped {
		t.Error("expected not tripped")
	}
}

func TestTrippedLocalSentinelTakesPrecedence(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "stop")
	if err := os.WriteFile(sentinel, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	c := New(fakeCloud{err: errors.New("cloud down")}, "m1", sentinel)
	tripped, err := c.Tripped(context.Background())
	if err != nil {
		t.Fatalf("tripped: %v", err)
	}
	if !tripped {
		t.Error("local sentinel should trip the switch regardless of cloud")
	}
}

func TestCloudErrorPropagates(t *testing.T) {
	c := New(fakeCloud{err: errors.New("cloud down")}, "m1", "")
	if _, err := c.Tripped(context.Background()); err == nil {
		t.Error("expected cloud error to propagate when no sentinel trips")
	}
}
