// ABOUTME: Machine operations: register (upsert), heartbeat, lookup, and kill switch.
// ABOUTME: Registration is idempotent on the unique machine name.

package store

import (
	"context"
	"fmt"
)

const machineColumns = `id, name, os, token_hash, last_seen, kill_switch, created_at`

// RegisterMachine inserts a machine or, if the name already exists, updates its
// OS and token hash. It returns the resulting machine row.
func (s *Store) RegisterMachine(ctx context.Context, name, os, tokenHash string) (Machine, error) {
	const q = `
INSERT INTO machines (name, os, token_hash)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET os = EXCLUDED.os, token_hash = EXCLUDED.token_hash
RETURNING ` + machineColumns
	row := s.pool.QueryRow(ctx, q, name, os, tokenHash)
	m, err := scanMachine(row)
	if err != nil {
		return Machine{}, fmt.Errorf("store: register machine: %w", err)
	}
	return m, nil
}

// GetMachineByName returns the machine with the given name.
func (s *Store) GetMachineByName(ctx context.Context, name string) (Machine, error) {
	const q = `SELECT ` + machineColumns + ` FROM machines WHERE name = $1`
	m, err := scanMachine(s.pool.QueryRow(ctx, q, name))
	if err != nil {
		return Machine{}, fmt.Errorf("store: get machine %q: %w", name, err)
	}
	return m, nil
}

// Heartbeat sets the machine's last_seen to now.
func (s *Store) Heartbeat(ctx context.Context, machineID string) error {
	const q = `UPDATE machines SET last_seen = now() WHERE id = $1`
	if _, err := s.pool.Exec(ctx, q, machineID); err != nil {
		return fmt.Errorf("store: heartbeat: %w", err)
	}
	return nil
}

// SetKillSwitch sets the machine's kill switch flag.
func (s *Store) SetKillSwitch(ctx context.Context, machineID string, on bool) error {
	const q = `UPDATE machines SET kill_switch = $2 WHERE id = $1`
	if _, err := s.pool.Exec(ctx, q, machineID, on); err != nil {
		return fmt.Errorf("store: set kill switch: %w", err)
	}
	return nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanMachine(r rowScanner) (Machine, error) {
	var m Machine
	if err := r.Scan(&m.ID, &m.Name, &m.OS, &m.TokenHash, &m.LastSeen, &m.KillSwitch, &m.CreatedAt); err != nil {
		return Machine{}, err
	}
	return m, nil
}
