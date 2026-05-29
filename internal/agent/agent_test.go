// ABOUTME: Tests for token hashing and machine registration.
// ABOUTME: Register is DB-backed (skips without TEST_DATABASE_URL); hashToken is pure.
package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/notpritam/beacon/internal/executor"
	"github.com/notpritam/beacon/internal/localaudit"
)

func TestHashTokenDeterministicAndHex(t *testing.T) {
	a := hashToken("secret")
	b := hashToken("secret")
	if a != b {
		t.Error("hashToken should be deterministic")
	}
	if hashToken("other") == a {
		t.Error("different tokens should hash differently")
	}
	if len(a) != 64 || strings.ContainsAny(a, "ghijklmnopqrstuvwxyz") {
		t.Errorf("expected 64-char lowercase hex, got %q", a)
	}
}

func newTestAgent(t *testing.T, st Store) *Agent {
	t.Helper()
	return New(st, executor.New(executor.DefaultConfig()), localaudit.New(t.TempDir()+"/audit.log"), "", Options{PollInterval: 10 * time.Millisecond})
}

func TestRegister(t *testing.T) {
	st := newAgentStore(t)
	a := newTestAgent(t, st)
	if err := a.Register(context.Background(), t.Name(), "darwin", "tok"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if a.machineID == "" {
		t.Error("machineID should be set after register")
	}
	m, err := st.MachineByName(context.Background(), t.Name())
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if m.TokenHash != hashToken("tok") {
		t.Error("stored token should be the hash, not the raw token")
	}
}
