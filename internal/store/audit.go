// ABOUTME: Append-only audit log operations for job lifecycle events.
// ABOUTME: ListAudit returns entries newest-first for display.

package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// AppendAudit records one audit event. jobID/machineID may be empty strings,
// which are stored as SQL NULL.
func (s *Store) AppendAudit(ctx context.Context, jobID, machineID, event string, detail json.RawMessage) error {
	const q = `
INSERT INTO audit_log (job_id, machine_id, event, detail)
VALUES (NULLIF($1,'')::uuid, NULLIF($2,'')::uuid, $3, $4)`
	if _, err := s.pool.Exec(ctx, q, jobID, machineID, event, detail); err != nil {
		return fmt.Errorf("store: append audit: %w", err)
	}
	return nil
}

// ListAudit returns up to limit audit entries, newest first.
func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	const q = `
SELECT id, COALESCE(job_id::text,''), COALESCE(machine_id::text,''), event, detail, at
FROM audit_log ORDER BY id DESC LIMIT $1`
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list audit: %w", err)
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var detail []byte
		if err := rows.Scan(&e.ID, &e.JobID, &e.MachineID, &e.Event, &detail, &e.At); err != nil {
			return nil, fmt.Errorf("store: scan audit: %w", err)
		}
		if detail != nil {
			e.Detail = json.RawMessage(detail)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate audit: %w", err)
	}
	return out, nil
}
