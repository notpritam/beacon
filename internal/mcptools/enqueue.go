// ABOUTME: The enqueuing MCP tools: run_command, read_file, write_file, list_dir.
// ABOUTME: Each enqueues a typed job and long-polls when the machine is online.

package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/notpritam/beacon/internal/store"
)

const createdByWingman = "wingman"

const payloadPathKey = "path"

func terminal(s store.JobStatus) bool {
	return s == store.JobDone || s == store.JobFailed || s == store.JobExpired
}

// enqueueAndWait inserts the job, returns immediately if the machine is offline,
// otherwise long-polls until the job reaches a terminal state or WaitTimeout.
func (t *Tools) enqueueAndWait(ctx context.Context, m store.Machine, jobType store.JobType, payload json.RawMessage) (JobOutcome, error) {
	job, err := t.store.EnqueueJob(ctx, m.ID, jobType, payload, 0, nil, createdByWingman)
	if err != nil {
		return JobOutcome{}, fmt.Errorf("mcptools: enqueue %s: %w", jobType, err)
	}
	if !t.online(m) {
		return JobOutcome{JobID: job.ID, Status: string(store.JobQueued), MachineOnline: false}, nil
	}

	deadline := time.Now().Add(t.opts.WaitTimeout)
	ticker := time.NewTicker(t.opts.PollInterval)
	defer ticker.Stop()
	for {
		got, err := t.store.GetJob(ctx, job.ID)
		if err != nil {
			return JobOutcome{}, fmt.Errorf("mcptools: poll job: %w", err)
		}
		if terminal(got.Status) {
			result, err := decodeResult(got.Result)
			if err != nil {
				return JobOutcome{}, err
			}
			return JobOutcome{JobID: job.ID, Status: string(got.Status), MachineOnline: true, Result: result}, nil
		}
		if time.Now().After(deadline) {
			return JobOutcome{JobID: job.ID, Status: string(got.Status), MachineOnline: true}, nil
		}
		select {
		case <-ctx.Done():
			return JobOutcome{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (t *Tools) resolve(ctx context.Context, name string) (store.Machine, error) {
	m, err := t.store.MachineByName(ctx, name)
	if err != nil {
		return store.Machine{}, fmt.Errorf("mcptools: machine %q: %w", name, err)
	}
	return m, nil
}

// RunCommand enqueues a shell job for the machine and waits for the result.
func (t *Tools) RunCommand(ctx context.Context, machine, cmd, cwd string, timeoutSecs int) (JobOutcome, error) {
	m, err := t.resolve(ctx, machine)
	if err != nil {
		return JobOutcome{}, err
	}
	payload, err := json.Marshal(map[string]any{"cmd": cmd, "cwd": cwd, "timeout_secs": timeoutSecs})
	if err != nil {
		return JobOutcome{}, fmt.Errorf("mcptools: run_command payload: %w", err)
	}
	return t.enqueueAndWait(ctx, m, store.JobShell, payload)
}

// ReadFile enqueues a read_file job for the machine and waits for the result.
func (t *Tools) ReadFile(ctx context.Context, machine, path string) (JobOutcome, error) {
	m, err := t.resolve(ctx, machine)
	if err != nil {
		return JobOutcome{}, err
	}
	payload, err := json.Marshal(map[string]string{payloadPathKey: path})
	if err != nil {
		return JobOutcome{}, fmt.Errorf("mcptools: read_file payload: %w", err)
	}
	return t.enqueueAndWait(ctx, m, store.JobReadFile, payload)
}

// WriteFile enqueues a write_file job for the machine and waits for the result.
func (t *Tools) WriteFile(ctx context.Context, machine, path, content string) (JobOutcome, error) {
	m, err := t.resolve(ctx, machine)
	if err != nil {
		return JobOutcome{}, err
	}
	payload, err := json.Marshal(map[string]string{payloadPathKey: path, "content": content})
	if err != nil {
		return JobOutcome{}, fmt.Errorf("mcptools: write_file payload: %w", err)
	}
	return t.enqueueAndWait(ctx, m, store.JobWriteFile, payload)
}

// ListDir enqueues a list_dir job for the machine and waits for the result.
func (t *Tools) ListDir(ctx context.Context, machine, path string) (JobOutcome, error) {
	m, err := t.resolve(ctx, machine)
	if err != nil {
		return JobOutcome{}, err
	}
	payload, err := json.Marshal(map[string]string{payloadPathKey: path})
	if err != nil {
		return JobOutcome{}, fmt.Errorf("mcptools: list_dir payload: %w", err)
	}
	return t.enqueueAndWait(ctx, m, store.JobListDir, payload)
}
