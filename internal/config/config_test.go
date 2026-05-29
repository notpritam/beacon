// ABOUTME: Tests for environment-based configuration loading and validation.
// ABOUTME: Uses t.Setenv so each case is isolated.
package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		wantErr   bool
		wantDB    string
		wantName  string
		wantToken string
	}{
		{
			name: "valid",
			env: map[string]string{
				"BEACON_DATABASE_URL":  "postgres://u:p@h:5432/db",
				"BEACON_MACHINE_NAME":  "test-mac",
				"BEACON_MACHINE_TOKEN": "secret",
			},
			wantDB:    "postgres://u:p@h:5432/db",
			wantName:  "test-mac",
			wantToken: "secret",
		},
		{
			name:    "missing database url",
			env:     map[string]string{"BEACON_MACHINE_NAME": "x", "BEACON_MACHINE_TOKEN": "y"},
			wantErr: true,
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
		})
	}
}
