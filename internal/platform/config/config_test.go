package config

import (
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
}

func TestLoad_UsesDefaultsWhenUnset(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.AppEnv != EnvDevelopment {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, EnvDevelopment)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestLoad_ReadsValuesFromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", EnvProduction)
	t.Setenv("PORT", "9000")
	t.Setenv("SHUTDOWN_TIMEOUT", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.IsProduction() {
		t.Errorf("IsProduction() = false, want true")
	}
	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want 9000", cfg.Port)
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", cfg.ShutdownTimeout)
	}
}

func TestLoad_RejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"unknown app env", "APP_ENV", "staging"},
		{"port is not a number", "PORT", "abc"},
		{"port is zero", "PORT", "0"},
		{"port is out of range", "PORT", "70000"},
		{"shutdown timeout has no unit", "SHUTDOWN_TIMEOUT", "10"},
		{"shutdown timeout is zero", "SHUTDOWN_TIMEOUT", "0s"},
		{"shutdown timeout is negative", "SHUTDOWN_TIMEOUT", "-5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(tt.key, tt.value)

			if _, err := Load(); err == nil {
				t.Fatalf("expected error for %s=%q, got nil", tt.key, tt.value)
			}
		})
	}
}

func TestLoad_ReportsAllErrorsAtOnce(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "staging")
	t.Setenv("PORT", "abc")
	t.Setenv("SHUTDOWN_TIMEOUT", "soon")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("expected a joined error, got %T", err)
	}
	if got := len(joined.Unwrap()); got != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", got, err)
	}
}
