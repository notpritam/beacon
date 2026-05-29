// ABOUTME: Job operations: enqueue, atomic claim (FOR UPDATE SKIP LOCKED), lifecycle
// ABOUTME: transitions (start/complete/fail), single-job lookup, and TTL expiry.

package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const jobColumns = `id, machine_id, type, payload, status, result, priority, ttl_at,
	created_by, created_at, claimed_at, finished_at`

// EnqueueJob inserts a new queued job for a machine and returns it.
func (s *Store) EnqueueJob(
	ctx context.Context,
	machineID string,
	jobType JobType,
	payload json.RawMessage,
	priority int,
	ttlAt *time.Time,
	createdBy string,
) (Job, error) {
	if !jobType.Valid() {
		return Job{}, fmt.Errorf("store: enqueue: invalid job type %q", jobType)
	}
	const q = `
INSERT INTO jobs (machine_id, type, payload, priority, ttl_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + jobColumns
	row := s.pool.QueryRow(ctx, q, machineID, string(jobType), payload, priority, ttlAt, createdBy)
	j, err := scanJob(row)
	if err != nil {
		return Job{}, fmt.Errorf("store: enqueue: %w", err)
	}
	return j, nil
}

// ClaimNextJob atomically claims the highest-priority, oldest queued job for a
// machine, transitioning it to "claimed". It returns nil if none are available.
func (s *Store) ClaimNextJob(ctx context.Context, machineID string) (*Job, error) {
	const q = `
UPDATE jobs SET status = 'claimed', claimed_at = now()
WHERE id = (
    SELECT id FROM jobs
    WHERE machine_id = $1 AND status = 'queued'
      AND (ttl_at IS NULL OR ttl_at > now())
    ORDER BY priority DESC, created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING ` + jobColumns
	j, err := scanJob(s.pool.QueryRow(ctx, q, machineID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: claim: %w", err)
	}
	return &j, nil
}

// StartJob marks a claimed job as running.
func (s *Store) StartJob(ctx context.Context, jobID string) error {
	return s.setStatus(ctx, jobID, JobRunning, nil, false)
}

// CompleteJob marks a job done and records its result.
func (s *Store) CompleteJob(ctx context.Context, jobID string, result json.RawMessage) error {
	return s.setStatus(ctx, jobID, JobDone, result, true)
}

// FailJob marks a job failed and records the failure detail as its result.
func (s *Store) FailJob(ctx context.Context, jobID string, result json.RawMessage) error {
	return s.setStatus(ctx, jobID, JobFailed, result, true)
}

func (s *Store) setStatus(ctx context.Context, jobID string, status JobStatus, result json.RawMessage, finished bool) error {
	const q = `
UPDATE jobs
SET status = $2,
    result = COALESCE($3, result),
    finished_at = CASE WHEN $4 THEN now() ELSE finished_at END
WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, jobID, string(status), result, finished)
	if err != nil {
		return fmt.Errorf("store: set status %q: %w", status, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: set status %q for %s: %w", status, jobID, ErrNotFound)
	}
	return nil
}

// GetJob returns a single job by id, or ErrNotFound if it does not exist.
func (s *Store) GetJob(ctx context.Context, jobID string) (Job, error) {
	const q = `SELECT ` + jobColumns + ` FROM jobs WHERE id = $1`
	j, err := scanJob(s.pool.QueryRow(ctx, q, jobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, fmt.Errorf("store: job %s: %w", jobID, ErrNotFound)
	}
	if err != nil {
		return Job{}, fmt.Errorf("store: get job %s: %w", jobID, err)
	}
	return j, nil
}

// ExpireDueJobs marks every queued job past its ttl_at as expired and returns
// the number affected.
func (s *Store) ExpireDueJobs(ctx context.Context) (int64, error) {
	const q = `
UPDATE jobs SET status = 'expired', finished_at = now()
WHERE status = 'queued' AND ttl_at IS NOT NULL AND ttl_at <= now()`
	tag, err := s.pool.Exec(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("store: expire jobs: %w", err)
	}
	return tag.RowsAffected(), nil
}

func scanJob(r rowScanner) (Job, error) {
	var (
		j       Job
		jType   string
		jStatus string
		payload []byte
		result  []byte
	)
	if err := r.Scan(
		&j.ID, &j.MachineID, &jType, &payload, &jStatus, &result,
		&j.Priority, &j.TTLAt, &j.CreatedBy, &j.CreatedAt, &j.ClaimedAt, &j.FinishedAt,
	); err != nil {
		return Job{}, err
	}
	j.Type = JobType(jType)
	j.Status = JobStatus(jStatus)
	j.Payload = json.RawMessage(payload)
	if result != nil {
		j.Result = json.RawMessage(result)
	}
	return j, nil
}
