// ABOUTME: The run loop: RunOnce claims and processes one job; Run loops with heartbeat.
// ABOUTME: Job execution failures are recorded (FailJob), not returned as loop errors.

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/notpritam/beacon/internal/localaudit"
	"github.com/notpritam/beacon/internal/store"
)

// errNotRegistered is returned when the loop runs before Register.
var errNotRegistered = errors.New("agent: not registered")

// RunOnce checks the kill switch, claims one job, and processes it. It returns
// (true, nil) if it processed a job, (false, nil) if there was nothing to do or
// the kill switch is tripped, and a non-nil error only on an infrastructure
// failure. A job whose execution fails is marked failed and still counts as work.
func (a *Agent) RunOnce(ctx context.Context) (bool, error) {
	if a.kill == nil {
		return false, errNotRegistered
	}
	tripped, err := a.kill.Tripped(ctx)
	if err != nil {
		return false, fmt.Errorf("agent: kill-switch check: %w", err)
	}
	if tripped {
		return false, nil
	}
	job, err := a.store.ClaimNextJob(ctx, a.machineID)
	if err != nil {
		return false, fmt.Errorf("agent: claim: %w", err)
	}
	if job == nil {
		return false, nil
	}
	if err := a.processJob(ctx, job); err != nil {
		return true, err
	}
	return true, nil
}

func (a *Agent) processJob(ctx context.Context, job *store.Job) error {
	a.appendAudit(ctx, job.ID, "claimed", nil)
	if err := a.store.StartJob(ctx, job.ID); err != nil {
		return fmt.Errorf("agent: start job %s: %w", job.ID, err)
	}

	result, execErr := a.exec.Execute(ctx, *job)
	if execErr != nil {
		if ctx.Err() != nil {
			// Shutdown/cancel is not a job failure; leave it running for the reaper.
			return fmt.Errorf("agent: job %s interrupted: %w", job.ID, ctx.Err())
		}
		failResult, err := json.Marshal(map[string]string{"error": execErr.Error()})
		if err != nil {
			failResult = []byte(`{"error":"execution failed"}`)
		}
		if err := a.store.FailJob(ctx, job.ID, failResult); err != nil {
			return fmt.Errorf("agent: fail job %s: %w", job.ID, err)
		}
		a.appendAudit(ctx, job.ID, "failed", failResult)
		a.log.Warn("job failed", "job", job.ID, "type", job.Type, "err", execErr)
		return nil
	}

	if err := a.store.CompleteJob(ctx, job.ID, result); err != nil {
		return fmt.Errorf("agent: complete job %s: %w", job.ID, err)
	}
	a.appendAudit(ctx, job.ID, "completed", nil)
	return nil
}

// appendAudit records an event to the cloud audit log and the local mirror.
// Audit is best-effort: failures are logged, never fatal to the job.
func (a *Agent) appendAudit(ctx context.Context, jobID, event string, detail json.RawMessage) {
	if err := a.store.AppendAudit(ctx, jobID, a.machineID, event, detail); err != nil {
		a.log.Warn("cloud audit failed", "job", jobID, "event", event, "err", err)
	}
	if err := a.audit.Append(localaudit.Entry{JobID: jobID, MachineID: a.machineID, Event: event, Detail: detail}); err != nil {
		a.log.Warn("local audit failed", "job", jobID, "event", event, "err", err)
	}
}

// Run drains the queue until ctx is cancelled, heartbeating on an interval and
// sleeping the poll interval when the queue is empty.
func (a *Agent) Run(ctx context.Context) error {
	if a.kill == nil {
		return errNotRegistered
	}
	hb := time.NewTicker(a.heartbeat)
	defer hb.Stop()
	a.beat(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-hb.C:
			a.beat(ctx)
		default:
		}

		did, err := a.RunOnce(ctx)
		if err != nil {
			a.log.Error("run once", "err", err)
		}
		if did && err == nil {
			continue // drain quickly only when work succeeded
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(a.poll):
		case <-hb.C:
			a.beat(ctx)
		}
	}
}

func (a *Agent) beat(ctx context.Context) {
	if err := a.store.Heartbeat(ctx, a.machineID); err != nil {
		a.log.Warn("heartbeat failed", "machine", a.machineID, "err", err)
	}
}
