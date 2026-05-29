// ABOUTME: Executor runs a store.Job on the local machine and returns its JSON result.
// ABOUTME: Dispatches by job type; shell/file ops are supported, screenshot/background deferred.

// Package executor performs the local side of a Beacon job: it runs shell
// commands and file operations and returns a JSON result.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/notpritam/beacon/internal/store"
)

// Config tunes execution limits.
type Config struct {
	// DefaultTimeout bounds a shell command when the payload does not specify one.
	DefaultTimeout time.Duration
	// MaxTimeout caps a payload-requested shell timeout.
	MaxTimeout time.Duration
	// MaxOutputBytes caps captured stdout and stderr (each) for a shell job.
	MaxOutputBytes int
	// MaxReadBytes caps the size of a file that read_file will return.
	MaxReadBytes int64
}

// DefaultConfig returns sensible execution limits.
func DefaultConfig() Config {
	return Config{
		DefaultTimeout: 60 * time.Second,
		MaxTimeout:     5 * time.Minute,
		MaxOutputBytes: 1 << 20,  // 1 MiB
		MaxReadBytes:   10 << 20, // 10 MiB
	}
}

// Executor runs jobs on the local machine.
type Executor struct {
	cfg Config
}

// New returns an Executor with the given configuration.
func New(cfg Config) *Executor {
	return &Executor{cfg: cfg}
}

// Execute runs the job and returns its result as JSON. A non-nil error means the
// job could not be executed (bad payload, missing file, timeout, unsupported
// type). A shell command that completes with a non-zero exit code is NOT an
// error — the exit code is carried in the result.
func (e *Executor) Execute(ctx context.Context, job store.Job) (json.RawMessage, error) {
	switch job.Type {
	case store.JobShell:
		return e.runShell(ctx, job.Payload)
	case store.JobReadFile:
		return e.readFile(job.Payload)
	case store.JobWriteFile:
		return e.writeFile(job.Payload)
	case store.JobListDir:
		return e.listDir(job.Payload)
	default:
		return nil, fmt.Errorf("executor: unsupported job type %q", job.Type)
	}
}
