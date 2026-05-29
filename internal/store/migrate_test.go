// ABOUTME: DB-backed test that the embedded migration creates the expected tables.
// ABOUTME: Exercises newTestStore; skips when TEST_DATABASE_URL is unset.
package store

import (
	"context"
	"testing"
)

func TestMigrateCreatesTables(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	for _, table := range []string{"machines", "jobs", "audit_log"} {
		var exists bool
		err := s.pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`,
			table).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %q should exist after migrate", table)
		}
	}
}
