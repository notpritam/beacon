// ABOUTME: Tools turns MCP tool calls into queued jobs and reads machine/job state.
// ABOUTME: Declares a small Store interface (the real store satisfies it) for testability.

// Package mcptools implements the Beacon MCP tool surface over the job store.
package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/notpritam/beacon/internal/store"
)

// Store is the subset of the database the tools use.
type Store interface {
	ListMachines(ctx context.Context) ([]store.Machine, error)
	MachineByName(ctx context.Context, name string) (store.Machine, error)
	EnqueueJob(ctx context.Context, machineID string, jobType store.JobType, payload json.RawMessage, priority int, ttlAt *time.Time, createdBy string) (store.Job, error)
	GetJob(ctx context.Context, jobID string) (store.Job, error)
}

// Options tunes online detection and long-poll behavior.
type Options struct {
	// OnlineThreshold is how recent a heartbeat must be for a machine to count as online.
	OnlineThreshold time.Duration
	// PollInterval is how often enqueueAndWait re-checks a job.
	PollInterval time.Duration
	// WaitTimeout bounds how long enqueueAndWait waits for an online machine's job.
	WaitTimeout time.Duration
}

// Tools implements the MCP tool surface.
type Tools struct {
	store Store
	opts  Options
}

// New returns Tools with sensible defaults applied to opts.
func New(st Store, opts Options) *Tools {
	if opts.OnlineThreshold <= 0 {
		opts.OnlineThreshold = 30 * time.Second
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 250 * time.Millisecond
	}
	if opts.WaitTimeout <= 0 {
		opts.WaitTimeout = 60 * time.Second
	}
	return &Tools{store: st, opts: opts}
}

// MachineInfo is the machine view returned to callers.
type MachineInfo struct {
	Name       string     `json:"name"`
	OS         string     `json:"os"`
	Online     bool       `json:"online"`
	LastSeen   *time.Time `json:"last_seen,omitempty"`
	KillSwitch bool       `json:"kill_switch"`
}

// JobOutcome is the result of enqueuing a job or fetching one.
type JobOutcome struct {
	JobID         string          `json:"job_id"`
	Status        string          `json:"status"`
	MachineOnline bool            `json:"machine_online"`
	Result        json.RawMessage `json:"result,omitempty"`
}

func (t *Tools) online(m store.Machine) bool {
	return m.LastSeen != nil && time.Since(*m.LastSeen) < t.opts.OnlineThreshold
}

func (t *Tools) toInfo(m store.Machine) MachineInfo {
	return MachineInfo{
		Name: m.Name, OS: m.OS, Online: t.online(m),
		LastSeen: m.LastSeen, KillSwitch: m.KillSwitch,
	}
}

// ListMachines returns all registered machines with online status.
func (t *Tools) ListMachines(ctx context.Context) ([]MachineInfo, error) {
	ms, err := t.store.ListMachines(ctx)
	if err != nil {
		return nil, fmt.Errorf("mcptools: list machines: %w", err)
	}
	out := make([]MachineInfo, 0, len(ms))
	for _, m := range ms {
		out = append(out, t.toInfo(m))
	}
	return out, nil
}

// MachineStatus returns the status of one machine by name.
func (t *Tools) MachineStatus(ctx context.Context, name string) (MachineInfo, error) {
	m, err := t.store.MachineByName(ctx, name)
	if err != nil {
		return MachineInfo{}, fmt.Errorf("mcptools: machine status: %w", err)
	}
	return t.toInfo(m), nil
}

// GetJob returns the current status and result of a job.
func (t *Tools) GetJob(ctx context.Context, jobID string) (JobOutcome, error) {
	j, err := t.store.GetJob(ctx, jobID)
	if err != nil {
		return JobOutcome{}, fmt.Errorf("mcptools: get job: %w", err)
	}
	return JobOutcome{JobID: j.ID, Status: string(j.Status), Result: j.Result}, nil
}
