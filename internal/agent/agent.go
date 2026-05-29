// ABOUTME: Agent registers the machine and holds the dependencies the run loop needs.
// ABOUTME: Declares small interfaces (Store/Executor/AuditMirror) so it is testable.

// Package agent runs the laptop side of Beacon: it registers the machine and
// drains its job queue, executing each job and reporting the result.
package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/notpritam/beacon/internal/killswitch"
	"github.com/notpritam/beacon/internal/localaudit"
	"github.com/notpritam/beacon/internal/store"
)

// Store is the subset of the database the agent uses. It also satisfies
// killswitch.CloudFlag via MachineKillSwitch.
type Store interface {
	RegisterMachine(ctx context.Context, name, os, tokenHash string) (store.Machine, error)
	Heartbeat(ctx context.Context, machineID string) error
	ClaimNextJob(ctx context.Context, machineID string) (*store.Job, error)
	StartJob(ctx context.Context, jobID string) error
	CompleteJob(ctx context.Context, jobID string, result json.RawMessage) error
	FailJob(ctx context.Context, jobID string, result json.RawMessage) error
	AppendAudit(ctx context.Context, jobID, machineID, event string, detail json.RawMessage) error
	MachineKillSwitch(ctx context.Context, machineID string) (bool, error)
}

// Executor runs a job and returns its JSON result.
type Executor interface {
	Execute(ctx context.Context, job store.Job) (json.RawMessage, error)
}

// AuditMirror appends an audit entry to the local mirror.
type AuditMirror interface {
	Append(entry localaudit.Entry) error
}

// Options configures the run loop.
type Options struct {
	// PollInterval is how long to wait when the queue is empty.
	PollInterval time.Duration
	// HeartbeatInterval is how often to update the machine's last_seen.
	HeartbeatInterval time.Duration
	// Logger receives loop diagnostics; defaults to slog.Default().
	Logger *slog.Logger
}

// Agent registers a machine and drains its job queue.
type Agent struct {
	store     Store
	exec      Executor
	audit     AuditMirror
	kill      *killswitch.Checker
	sentinel  string
	machineID string
	name      string
	poll      time.Duration
	heartbeat time.Duration
	log       *slog.Logger
}

// New returns an Agent. sentinel is the local kill-switch file path (empty disables it).
func New(st Store, exec Executor, audit AuditMirror, sentinel string, opts Options) *Agent {
	if opts.PollInterval <= 0 {
		opts.PollInterval = 2 * time.Second
	}
	if opts.HeartbeatInterval <= 0 {
		opts.HeartbeatInterval = 15 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Agent{
		store: st, exec: exec, audit: audit, sentinel: sentinel,
		poll: opts.PollInterval, heartbeat: opts.HeartbeatInterval, log: opts.Logger,
	}
}

// Register records this machine (hashing the token) and prepares the kill switch.
func (a *Agent) Register(ctx context.Context, name, osName, token string) error {
	m, err := a.store.RegisterMachine(ctx, name, osName, hashToken(token))
	if err != nil {
		return fmt.Errorf("agent: register: %w", err)
	}
	a.machineID = m.ID
	a.name = m.Name
	a.kill = killswitch.New(a.store, a.machineID, a.sentinel)
	return nil
}

// MachineID returns this machine's id (set by Register), or "" if not registered.
func (a *Agent) MachineID() string {
	return a.machineID
}

// hashToken returns the hex-encoded SHA-256 of the token.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
