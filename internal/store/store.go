// ABOUTME: Store is the single typed gateway to Beacon's Postgres database.
// ABOUTME: Wraps a pgxpool.Pool; all DB access in the codebase goes through it.

package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// maxConns caps each process's pool so several Beacon processes (agent, mcp,
// dashboard) and tests fit within a shared/pooled Postgres connection limit —
// e.g. Supabase's free-tier session pooler caps total clients at 15.
const maxConns = 4

// Store provides database operations for machines, jobs, and audit entries.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a connection pool to the database at databaseURL.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse config: %w", err)
	}
	if cfg.MaxConns > maxConns {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the connection pool.
func (s *Store) Close() {
	s.pool.Close()
}
