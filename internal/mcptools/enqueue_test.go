// ABOUTME: DB-backed tests for the enqueuing MCP tools (offline-returns-queued, online-completes).
// ABOUTME: The online test runs a goroutine that claims+completes the job, simulating the agent.
package mcptools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/notpritam/beacon/internal/store"
)

func TestRunCommandOfflineReturnsQueued(t *testing.T) {
	st := newToolsStore(t)
	tools := New(st, Options{})
	ctx := context.Background()
	if _, err := st.RegisterMachine(ctx, t.Name(), "darwin", "h"); err != nil {
		t.Fatalf("register: %v", err)
	}
	out, err := tools.RunCommand(ctx, t.Name(), "echo hi", "", 0)
	if err != nil {
		t.Fatalf("run_command: %v", err)
	}
	if out.MachineOnline {
		t.Error("machine should be offline")
	}
	if out.Status != string(store.JobQueued) || out.JobID == "" {
		t.Errorf("expected queued job id, got %+v", out)
	}
}

func TestRunCommandOnlineCompletes(t *testing.T) {
	st := newToolsStore(t)
	tools := New(st, Options{PollInterval: 20 * time.Millisecond, WaitTimeout: 5 * time.Second})
	ctx := context.Background()
	m, err := st.RegisterMachine(ctx, t.Name(), "darwin", "h")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := st.Heartbeat(ctx, m.ID); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			job, err := st.ClaimNextJob(ctx, m.ID)
			if err != nil {
				return
			}
			if job != nil {
				_ = st.CompleteJob(ctx, job.ID, json.RawMessage(`{"stdout":"hi\n","exit_code":0}`))
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	out, err := tools.RunCommand(ctx, t.Name(), "echo hi", "", 0)
	<-done
	if err != nil {
		t.Fatalf("run_command: %v", err)
	}
	if !out.MachineOnline || out.Status != string(store.JobDone) {
		t.Fatalf("expected online+done, got %+v", out)
	}
	obj, ok := out.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %T", out.Result)
	}
	if stdout, _ := obj["stdout"].(string); stdout == "" {
		t.Errorf("expected stdout in result, got %v", obj)
	}
}

func TestReadFileEnqueuesCorrectPayload(t *testing.T) {
	st := newToolsStore(t)
	tools := New(st, Options{})
	ctx := context.Background()
	if _, err := st.RegisterMachine(ctx, t.Name(), "darwin", "h"); err != nil {
		t.Fatalf("register: %v", err)
	}
	out, err := tools.ReadFile(ctx, t.Name(), "/tmp/x")
	if err != nil {
		t.Fatalf("read_file: %v", err)
	}
	job, err := st.GetJob(ctx, out.JobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.Type != store.JobReadFile {
		t.Errorf("type = %q, want read_file", job.Type)
	}
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if p.Path != "/tmp/x" {
		t.Errorf("path = %q, want /tmp/x", p.Path)
	}
}
