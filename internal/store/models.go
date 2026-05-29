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
	JobBackground JobType = "background"
)

// Valid reports whether t is a known job type.
func (t JobType) Valid() bool {
	switch t {
	case JobShell, JobReadFile, JobWriteFile, JobListDir, JobScreenshot, JobBackground:
		return true
	default:
		return false
	}
}

// Machine is a registered laptop that runs jobs.
type Machine struct {
	ID         string
	Name       string
	OS         string
	TokenHash  string
	LastSeen   *time.Time
	KillSwitch bool
	CreatedAt  time.Time
}

// Job is a unit of work targeted at a machine.
type Job struct {
	ID         string
	MachineID  string
	Type       JobType
	Payload    json.RawMessage
	Status     JobStatus
	Result     json.RawMessage
	Priority   int
	TTLAt      *time.Time
	CreatedBy  string
	CreatedAt  time.Time
	ClaimedAt  *time.Time
	FinishedAt *time.Time
}

// AuditEntry is one append-only record of a job lifecycle event.
type AuditEntry struct {
	ID        int64
	JobID     string
	MachineID string
	Event     string
	Detail    json.RawMessage
	At        time.Time
}
