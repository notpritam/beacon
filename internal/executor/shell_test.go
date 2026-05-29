// ABOUTME: Tests for shell job execution — output capture, exit codes, cwd, and timeout.
// ABOUTME: Pure tests; no database required.
package executor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func runShellJob(t *testing.T, e *Executor, payload string) shellResult {
	t.Helper()
	job := store.Job{Type: store.JobShell, Payload: json.RawMessage(payload)}
	raw, err := e.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var res shellResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return res
}

func jsonQuote(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestShellEcho(t *testing.T) {
	res := runShellJob(t, New(DefaultConfig()), `{"cmd":"echo hello"}`)
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("stdout = %q, want it to contain hello", res.Stdout)
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0", res.ExitCode)
	}
}

func TestShellExitCode(t *testing.T) {
	res := runShellJob(t, New(DefaultConfig()), `{"cmd":"exit 3"}`)
	if res.ExitCode != 3 {
		t.Errorf("exit = %d, want 3", res.ExitCode)
	}
}

func TestShellStderr(t *testing.T) {
	res := runShellJob(t, New(DefaultConfig()), `{"cmd":"echo oops 1>&2"}`)
	if !strings.Contains(res.Stderr, "oops") {
		t.Errorf("stderr = %q, want it to contain oops", res.Stderr)
	}
}

func TestShellCwd(t *testing.T) {
	dir := t.TempDir()
	payload := `{"cmd":"pwd","cwd":` + jsonQuote(t, dir) + `}`
	res := runShellJob(t, New(DefaultConfig()), payload)
	if !strings.Contains(res.Stdout, dir) {
		t.Errorf("stdout = %q, want it to contain %q", res.Stdout, dir)
	}
}

func TestShellEmptyCmd(t *testing.T) {
	e := New(DefaultConfig())
	job := store.Job{Type: store.JobShell, Payload: json.RawMessage(`{"cmd":""}`)}
	if _, err := e.Execute(context.Background(), job); err == nil {
		t.Error("expected error for empty cmd")
	}
}

func TestShellTimeout(t *testing.T) {
	e := New(DefaultConfig())
	job := store.Job{Type: store.JobShell, Payload: json.RawMessage(`{"cmd":"sleep 5","timeout_secs":1}`)}
	_, err := e.Execute(context.Background(), job)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}
