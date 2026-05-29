// ABOUTME: Machine operations: register (upsert), heartbeat, lookup, and kill switch.
// ABOUTME: Registration is idempotent on the unique machine name.

package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

// MachineByName returns the machine with the given name, or ErrNotFound if absent.
func (s *Store) MachineByName(ctx context.Context, name string) (Machine, error) {
	const q = `SELECT ` + machineColumns + ` FROM machines WHERE name = $1`
	m, err := scanMachine(s.pool.QueryRow(ctx, q, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return Machine{}, fmt.Errorf("store: machine %q: %w", name, ErrNotFound)
	}
	if err != nil {
		return Machine{}, fmt.Errorf("store: get machine %q: %w", name, err)
	}
	return m, nil
}

// Heartbeat sets the machine's last_seen to now.
func (s *Store) Heartbeat(ctx context.Context, machineID string) error {
	const q = `UPDATE machines SET last_seen = now() WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, machineID)
	if err != nil {
		return fmt.Errorf("store: heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: heartbeat %s: %w", machineID, ErrNotFound)
	}
	return nil
}

// SetKillSwitch sets the machine's kill switch flag.
func (s *Store) SetKillSwitch(ctx context.Context, machineID string, on bool) error {
	const q = `UPDATE machines SET kill_switch = $2 WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, machineID, on)
	if err != nil {
		return fmt.Errorf("store: set kill switch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store: set kill switch %s: %w", machineID, ErrNotFound)
	}
	return nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// ListMachines returns all registered machines, newest first.
func (s *Store) ListMachines(ctx context.Context) ([]Machine, error) {
	const q = `SELECT ` + machineColumns + ` FROM machines ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: list machines: %w", err)
	}
	defer rows.Close()
	var out []Machine
	for rows.Next() {
		m, err := scanMachine(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan machine: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate machines: %w", err)
	}
	return out, nil
}

func scanMachine(r rowScanner) (Machine, error) {
	var m Machine
	if err := r.Scan(&m.ID, &m.Name, &m.OS, &m.TokenHash, &m.LastSeen, &m.KillSwitch, &m.CreatedAt); err != nil {
		return Machine{}, err
	}
	return m, nil
}
