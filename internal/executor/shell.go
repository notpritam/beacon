// ABOUTME: Shell job execution: runs `sh -c <cmd>` under a bounded timeout and captures output.
// ABOUTME: A non-zero exit code is a successful run reported in the result, not an error.

package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

type shellPayload struct {
	Cmd         string `json:"cmd"`
	Cwd         string `json:"cwd"`
	TimeoutSecs int    `json:"timeout_secs"`
}

type shellResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func (e *Executor) runShell(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var p shellPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("executor: shell payload: %w", err)
	}
	if p.Cmd == "" {
		return nil, fmt.Errorf("executor: shell: empty cmd")
	}

	timeout := e.cfg.DefaultTimeout
	if p.TimeoutSecs > 0 {
		timeout = time.Duration(p.TimeoutSecs) * time.Second
		if timeout > e.cfg.MaxTimeout {
			timeout = e.cfg.MaxTimeout
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "sh", "-c", p.Cmd) //nolint:gosec // executing arbitrary operator-supplied commands is the explicit purpose of this agent
	cmd.Dir = p.Cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runCtx.Err() != nil {
		return nil, fmt.Errorf("executor: shell timed out after %s: %w", timeout, runCtx.Err())
	}

	res := shellResult{
		Stdout: capString(stdout.String(), e.cfg.MaxOutputBytes),
		Stderr: capString(stderr.String(), e.cfg.MaxOutputBytes),
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			return nil, fmt.Errorf("executor: shell run: %w", runErr)
		}
		res.ExitCode = exitErr.ExitCode()
	}

	out, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("executor: shell marshal: %w", err)
	}
	return out, nil
}

// capString truncates s to at most maxBytes bytes (maxBytes <= 0 means no limit).
func capString(s string, maxBytes int) string {
	if maxBytes > 0 && len(s) > maxBytes {
		return s[:maxBytes]
	}
	return s
}
