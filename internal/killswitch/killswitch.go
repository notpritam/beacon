// ABOUTME: Checker reports whether the agent's kill switch is engaged.
// ABOUTME: A local sentinel file takes precedence over the cloud machines.kill_switch flag.

// Package killswitch decides whether the agent must stop claiming and executing
// jobs, from a local sentinel file or the cloud kill-switch flag.
package killswitch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// CloudFlag reads the cloud-side kill-switch flag for a machine.
type CloudFlag interface {
	MachineKillSwitch(ctx context.Context, machineID string) (bool, error)
}

// Checker reports whether the kill switch is engaged.
type Checker struct {
	cloud     CloudFlag
	machineID string
	sentinel  string
}

// New returns a Checker. sentinel is a local file path whose existence forces the
// switch on (empty disables the local check); cloud reads the machine's flag.
func New(cloud CloudFlag, machineID, sentinel string) *Checker {
	return &Checker{cloud: cloud, machineID: machineID, sentinel: sentinel}
}

// Tripped reports whether the agent must stop. The local sentinel file is checked
// first and short-circuits (so a disconnected machine can still be hard-stopped);
// otherwise the cloud flag is consulted.
func (c *Checker) Tripped(ctx context.Context) (bool, error) {
	if c.sentinel != "" {
		_, err := os.Stat(c.sentinel)
		switch {
		case err == nil:
			return true, nil
		case errors.Is(err, fs.ErrNotExist):
			// not tripped locally; fall through to the cloud flag
		default:
			return false, fmt.Errorf("killswitch: stat sentinel: %w", err)
		}
	}
	on, err := c.cloud.MachineKillSwitch(ctx, c.machineID)
	if err != nil {
		return false, fmt.Errorf("killswitch: cloud flag: %w", err)
	}
	return on, nil
}
