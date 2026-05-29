// ABOUTME: Loads Beacon configuration from environment variables and validates it.
// ABOUTME: Shared by the agent, the MCP server, and beaconctl.

// Package config loads and validates Beacon settings from environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds the settings every Beacon binary needs to reach the database
// and identify this machine.
type Config struct {
	// DatabaseURL is the Postgres connection string (Supabase or local).
	DatabaseURL string
	// MachineName is the human label for this machine.
	MachineName string
	// MachineToken is the per-machine secret presented for authentication.
	MachineToken string
}

// Load reads configuration from the environment and validates required fields.
func Load() (Config, error) {
	c := Config{
		DatabaseURL:  os.Getenv("BEACON_DATABASE_URL"),
		MachineName:  os.Getenv("BEACON_MACHINE_NAME"),
		MachineToken: os.Getenv("BEACON_MACHINE_TOKEN"),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: BEACON_DATABASE_URL is required")
	}
	return c, nil
}
