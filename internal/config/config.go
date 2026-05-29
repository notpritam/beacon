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
	// WingmanToken is the bearer token the MCP server requires from callers.
	WingmanToken string
	// MCPAddr is the MCP server listen address (default ":8080").
	MCPAddr string
	// DashboardToken gates the live-view dashboard. Empty disables the dashboard.
	DashboardToken string
	// DashboardAddr is the live-view dashboard listen address (default ":8081").
	DashboardAddr string
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
	c.WingmanToken = os.Getenv("BEACON_WINGMAN_TOKEN")
	c.MCPAddr = os.Getenv("BEACON_MCP_ADDR")
	if c.MCPAddr == "" {
		c.MCPAddr = ":8080"
	}
	c.DashboardToken = os.Getenv("BEACON_DASHBOARD_TOKEN")
	c.DashboardAddr = os.Getenv("BEACON_DASHBOARD_ADDR")
	if c.DashboardAddr == "" {
		c.DashboardAddr = ":8081"
	}
	return c, nil
}
