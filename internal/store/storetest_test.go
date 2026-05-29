// ABOUTME: Test helper that connects to TEST_DATABASE_URL, migrates, and truncates tables.
// ABOUTME: Tests skip cleanly when TEST_DATABASE_URL is unset so the gate stays green.
package store

import (
	"context"
	"os"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-backed test")
	}
	ctx := context.Background()
	s, err := New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := s.pool.Exec(ctx,
		`TRUNCATE audit_log, jobs, machines RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}
