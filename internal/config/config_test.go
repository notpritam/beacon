// ABOUTME: Tests for environment-based configuration loading and validation.
// ABOUTME: Uses t.Setenv so each case is isolated.
package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantErr     bool
		wantDB      string
		wantName    string
		wantToken   string
		wantWingman string
		wantAddr    string
	}{
		{
			name: "valid",
			env: map[string]string{
				"BEACON_DATABASE_URL":  "postgres://u:p@h:5432/db",
				"BEACON_MACHINE_NAME":  "test-mac",
				"BEACON_MACHINE_TOKEN": "secret",
				"BEACON_WINGMAN_TOKEN": "wtok",
				"BEACON_MCP_ADDR":      ":9000",
			},
			wantDB:      "postgres://u:p@h:5432/db",
			wantName:    "test-mac",
			wantToken:   "secret",
			wantWingman: "wtok",
			wantAddr:    ":9000",
		},
		{
			name:    "missing database url",
			env:     map[string]string{"BEACON_MACHINE_NAME": "x", "BEACON_MACHINE_TOKEN": "y"},
			wantErr: true,
		},
		{
			name:     "default mcp addr",
			env:      map[string]string{"BEACON_DATABASE_URL": "postgres://u:p@h:5432/db"},
			wantDB:   "postgres://u:p@h:5432/db",
			wantAddr: ":8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.DatabaseURL != tt.wantDB {
				t.Errorf("DatabaseURL = %q, want %q", got.DatabaseURL, tt.wantDB)
			}
			if got.MachineName != tt.wantName {
				t.Errorf("MachineName = %q, want %q", got.MachineName, tt.wantName)
			}
			if got.MachineToken != tt.wantToken {
				t.Errorf("MachineToken = %q, want %q", got.MachineToken, tt.wantToken)
			}
			if got.WingmanToken != tt.wantWingman {
				t.Errorf("WingmanToken = %q, want %q", got.WingmanToken, tt.wantWingman)
			}
			if got.MCPAddr != tt.wantAddr {
				t.Errorf("MCPAddr = %q, want %q", got.MCPAddr, tt.wantAddr)
			}
		})
	}
}
