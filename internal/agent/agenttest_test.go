// ABOUTME: Test helper that connects to TEST_DATABASE_URL and migrates; skips when unset.
// ABOUTME: Tests use unique machine names for isolation (no global truncate needed).
package agent

import (
	"context"
	"os"
	"testing"

	"github.com/notpritam/beacon/internal/store"
)

func newAgentStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-backed test")
	}
	ctx := context.Background()
	s, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}
