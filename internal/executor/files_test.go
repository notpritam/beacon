// ABOUTME: Tests for read_file/write_file/list_dir job execution against a temp dir.
// ABOUTME: Pure tests; no database required.
package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func execJob(t *testing.T, e *Executor, jt store.JobType, payload string) json.RawMessage {
	t.Helper()
	raw, err := e.Execute(context.Background(), store.Job{Type: jt, Payload: json.RawMessage(payload)})
	if err != nil {
		t.Fatalf("execute %s: %v", jt, err)
	}
	return raw
}

func pathJSON(t *testing.T, path string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(f, []byte("hi there"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	raw := execJob(t, New(DefaultConfig()), store.JobReadFile, pathJSON(t, f))
	var res readResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res.Content != "hi there" {
		t.Errorf("content = %q, want %q", res.Content, "hi there")
	}
}

func TestReadFileTooLarge(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(f, []byte("0123456789"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfg := DefaultConfig()
	cfg.MaxReadBytes = 5
	e := New(cfg)
	if _, err := e.Execute(context.Background(), store.Job{Type: store.JobReadFile, Payload: json.RawMessage(pathJSON(t, f))}); err == nil {
		t.Error("expected error for oversized file")
	}
}

func TestReadFileMissing(t *testing.T) {
	e := New(DefaultConfig())
	missing := filepath.Join(t.TempDir(), "nope.txt")
	if _, err := e.Execute(context.Background(), store.Job{Type: store.JobReadFile, Payload: json.RawMessage(pathJSON(t, missing))}); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.txt")
	payload, err := json.Marshal(map[string]string{"path": f, "content": "written!"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := execJob(t, New(DefaultConfig()), store.JobWriteFile, string(payload))
	var res writeResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res.BytesWritten != len("written!") {
		t.Errorf("bytes = %d, want %d", res.BytesWritten, len("written!"))
	}
	got, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	if string(got) != "written!" {
		t.Errorf("file = %q, want %q", got, "written!")
	}
}

func TestListDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}
	raw := execJob(t, New(DefaultConfig()), store.JobListDir, pathJSON(t, dir))
	var res listResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(res.Entries))
	}
	byName := map[string]dirEntry{}
	for _, ent := range res.Entries {
		byName[ent.Name] = ent
	}
	if !byName["sub"].IsDir {
		t.Error("sub should be a dir")
	}
	if byName["a.txt"].IsDir {
		t.Error("a.txt should not be a dir")
	}
}
