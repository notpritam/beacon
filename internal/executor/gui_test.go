// ABOUTME: Pure tests for GUI action validation (no system calls / permissions needed).
// ABOUTME: Real pointer/keyboard/app behavior requires macOS Accessibility and is not unit-tested.
package executor

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func guiExec(t *testing.T, payload string) error {
	t.Helper()
	e := New(DefaultConfig())
	_, err := e.Execute(context.Background(), store.Job{Type: store.JobGUI, Payload: json.RawMessage(payload)})
	return err
}

func TestGUIUnsupportedAction(t *testing.T) {
	if runtime.GOOS != osDarwin {
		t.Skip("gui is macOS-only")
	}
	if err := guiExec(t, `{"action":"frobnicate"}`); err == nil {
		t.Error("expected error for unsupported gui action")
	}
}

func TestGUIBadPayload(t *testing.T) {
	if runtime.GOOS != osDarwin {
		t.Skip("gui is macOS-only")
	}
	if err := guiExec(t, `not json`); err == nil {
		t.Error("expected error for invalid gui payload")
	}
}

func TestGUIValidationErrors(t *testing.T) {
	if runtime.GOOS != osDarwin {
		t.Skip("gui is macOS-only")
	}
	cases := []string{
		`{"action":"type","text":""}`,       // empty text
		`{"action":"key","key":""}`,         // empty key
		`{"action":"open_app","app":""}`,    // empty app
		`{"action":"hotkey","combo":""}`,    // empty combo
		`{"action":"hotkey","combo":"x+y"}`, // unknown modifier
	}
	for _, c := range cases {
		if err := guiExec(t, c); err == nil {
			t.Errorf("expected error for payload %s", c)
		}
	}
}
