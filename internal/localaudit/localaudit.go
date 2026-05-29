// ABOUTME: Logger appends audit events as JSON lines to a local file (append-only mirror).
// ABOUTME: Independent of the database so the local trail survives cloud-side tampering.

// Package localaudit writes an append-only local mirror of job audit events.
package localaudit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Entry is one audit record written to the local mirror.
type Entry struct {
	At        time.Time       `json:"at"`
	JobID     string          `json:"job_id"`
	MachineID string          `json:"machine_id"`
	Event     string          `json:"event"`
	Detail    json.RawMessage `json:"detail,omitempty"`
}

// Logger appends audit entries to a file. It is safe for concurrent use.
type Logger struct {
	path string
	mu   sync.Mutex
}

// New returns a Logger that appends to the file at path.
func New(path string) *Logger {
	return &Logger{path: path}
}

// Append writes one entry as a JSON line. If the entry has no timestamp, the
// current time is stamped. The file is created if it does not exist.
func (l *Logger) Append(entry Entry) error {
	if entry.At.IsZero() {
		entry.At = time.Now()
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("localaudit: marshal: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // appending to an agent-owned audit file
	if err != nil {
		return fmt.Errorf("localaudit: open: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("localaudit: write: %w", err)
	}
	return nil
}
