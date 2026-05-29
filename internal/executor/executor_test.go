// ABOUTME: Tests for Executor dispatch — unsupported job types must error.
// ABOUTME: Per-operation behavior is tested in shell_test.go (file ops arrive in Task 2).
package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func TestExecuteUnsupportedType(t *testing.T) {
	e := New(DefaultConfig())
	job := store.Job{Type: store.JobScreenshot, Payload: json.RawMessage(`{}`)}
	if _, err := e.Execute(context.Background(), job); err == nil {
		t.Fatal("expected error for unsupported job type, got nil")
	}
}
