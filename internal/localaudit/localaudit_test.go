// ABOUTME: Tests for the append-only local audit mirror.
// ABOUTME: Writes to a temp file and reads the JSONL back; no database.
package localaudit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendWritesJSONLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l := New(path)

	if err := l.Append(Entry{JobID: "j1", MachineID: "m1", Event: "created"}); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := l.Append(Entry{JobID: "j1", MachineID: "m1", Event: "done"}); err != nil {
		t.Fatalf("append 2: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	var events []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("unmarshal line: %v", err)
		}
		if e.At.IsZero() {
			t.Error("entry should have a timestamp")
		}
		events = append(events, e.Event)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(events) != 2 || events[0] != "created" || events[1] != "done" {
		t.Errorf("events = %v, want [created done]", events)
	}
}
