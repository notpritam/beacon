// ABOUTME: Domain models for machines, jobs, and audit entries plus their enums.
// ABOUTME: These types are the typed contract every other slice uses.

// Package store is the typed gateway to Beacon's Postgres database.
package store

import (
	"encoding/json"
	"time"
)

// JobStatus is the lifecycle state of a job.
type JobStatus string

// Job lifecycle states.
const (
	JobQueued  JobStatus = "queued"
	JobClaimed JobStatus = "claimed"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
	JobExpired JobStatus = "expired"
)

// Valid reports whether s is a known job status.
func (s JobStatus) Valid() bool {
	switch s {
	case JobQueued, JobClaimed, JobRunning, JobDone, JobFailed, JobExpired:
		return true
	default:
		return false
	}
}

// JobType is the kind of work a job represents.
type JobType string

// Supported job types.
const (
	JobShell      JobType = "shell"
	JobReadFile   JobType = "read_file"
	JobWriteFile  JobType = "write_file"
	JobListDir    JobType = "list_dir"
	JobScreenshot JobType = "screenshot"
	JobGUI        JobType = "gui"
	JobBackground JobType = "background"
)

// Valid reports whether t is a known job type.
func (t JobType) Valid() bool {
	switch t {
	case JobShell, JobReadFile, JobWriteFile, JobListDir, JobScreenshot, JobGUI, JobBackground:
		return true
	default:
		return false
	}
}

// Machine is a registered laptop that runs jobs.
type Machine struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	OS         string     `json:"os"`
	TokenHash  string     `json:"token_hash"`
	LastSeen   *time.Time `json:"last_seen"`
	KillSwitch bool       `json:"kill_switch"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Job is a unit of work targeted at a machine.
type Job struct {
	ID         string          `json:"id"`
	MachineID  string          `json:"machine_id"`
	Type       JobType         `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Status     JobStatus       `json:"status"`
	Result     json.RawMessage `json:"result"`
	Priority   int             `json:"priority"`
	TTLAt      *time.Time      `json:"ttl_at"`
	CreatedBy  string          `json:"created_by"`
	CreatedAt  time.Time       `json:"created_at"`
	ClaimedAt  *time.Time      `json:"claimed_at"`
	FinishedAt *time.Time      `json:"finished_at"`
}

// AuditEntry is one append-only record of a job lifecycle event.
type AuditEntry struct {
	ID        int64           `json:"id"`
	JobID     string          `json:"job_id"`
	MachineID string          `json:"machine_id"`
	Event     string          `json:"event"`
	Detail    json.RawMessage `json:"detail"`
	At        time.Time       `json:"at"`
}
