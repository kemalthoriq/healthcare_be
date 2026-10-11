package config

import (
	"strings"
	"testing"
	"time"
)

const testDatabaseURL = "postgres://user:pass@localhost:5433/db?sslmode=disable"

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("SEED_ADMIN_NAME", "")
	t.Setenv("SEED_ADMIN_EMAIL", "")
	t.Setenv("SEED_ADMIN_PASSWORD", "")
	t.Setenv("FRONTEND_ORIGIN", "")
	t.Setenv("SESSION_TTL", "")
	t.Setenv("EMAIL_ENABLED", "")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("TRUSTED_PROXY_HOPS", "")
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	t.Setenv("CLOUDINARY_CLOUD_NAME", "")
	t.Setenv("CLOUDINARY_API_KEY", "")
	t.Setenv("CLOUDINARY_API_SECRET", "")
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	clearEnv(t)
	t.Setenv("DATABASE_URL", testDatabaseURL)
}

func TestLoad_UsesDefaultsWhenUnset(t *testing.T) {
	setRequiredEnv(t)

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
	if cfg.DBMaxConns != 5 {
		t.Errorf("DBMaxConns = %d, want 5", cfg.DBMaxConns)
	}
	if cfg.FrontendOrigin != "http://localhost:3000" {
		t.Errorf("FrontendOrigin = %q, want http://localhost:3000", cfg.FrontendOrigin)
	}
	if cfg.SessionTTL != 168*time.Hour {
		t.Errorf("SessionTTL = %v, want 168h", cfg.SessionTTL)
	}
}

func TestLoad_ReadsValuesFromEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_ENV", EnvProduction)
	t.Setenv("PORT", "9000")
	t.Setenv("SHUTDOWN_TIMEOUT", "30s")
	t.Setenv("DB_MAX_CONNS", "10")
	t.Setenv("TURNSTILE_SECRET_KEY", "1x0000000000000000000000000000000AA")

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
	if cfg.DatabaseURL != testDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, testDatabaseURL)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("DBMaxConns = %d, want 10", cfg.DBMaxConns)
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
		{"max conns is not a number", "DB_MAX_CONNS", "many"},
		{"max conns is zero", "DB_MAX_CONNS", "0"},
		{"max conns is too large", "DB_MAX_CONNS", "51"},
		{"seed email without password", "SEED_ADMIN_EMAIL", "admin@lumiere.test"},
		{"seed password without email", "SEED_ADMIN_PASSWORD", "rahasia-sekali-123"},
		{"origin without scheme", "FRONTEND_ORIGIN", "localhost:3000"},
		{"origin with path", "FRONTEND_ORIGIN", "https://lumiere.vercel.app/admin"},
		{"session ttl without unit", "SESSION_TTL", "168"},
		{"session ttl is zero", "SESSION_TTL", "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tt.key, tt.value)

			if _, err := Load(); err == nil {
				t.Fatalf("expected error for %s=%q, got nil", tt.key, tt.value)
			}
		})
	}
}

func TestLoad_SeedAdmin(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{"valid seed", "admin@lumiere.test", "rahasia-sekali-123", false},
		{"invalid email", "bukan-email", "rahasia-sekali-123", true},
		{"short password", "admin@lumiere.test", "pendek", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("SEED_ADMIN_EMAIL", tt.email)
			t.Setenv("SEED_ADMIN_PASSWORD", tt.password)

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && cfg.SeedAdminName != "Admin" {
				t.Errorf("SeedAdminName = %q, want default Admin", cfg.SeedAdminName)
			}
		})
	}
}

func TestLoad_RequiresDatabaseURL(t *testing.T) {
	clearEnv(t)

	if _, err := Load(); err == nil {
		t.Fatal("expected error when DATABASE_URL is missing, got nil")
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
	if got := len(joined.Unwrap()); got != 4 {
		t.Fatalf("expected 4 errors, got %d: %v", got, err)
	}
}

func TestLoad_Email(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	if err != nil || cfg.EmailEnabled || cfg.SMTPPort != 587 {
		t.Fatalf("defaults: enabled = %v, port = %d, err = %v", cfg.EmailEnabled, cfg.SMTPPort, err)
	}

	t.Setenv("EMAIL_ENABLED", "true")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "SMTP_HOST") || !strings.Contains(err.Error(), "SMTP_FROM") {
		t.Fatalf("enabled without host/from: err = %v", err)
	}

	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("SMTP_FROM", "Lumière Clinic <noreply@lumiere.test>")
	cfg, err = Load()
	if err != nil || !cfg.EmailEnabled || cfg.SMTPPort != 1025 || cfg.SMTPUser != "" {
		t.Fatalf("mailpit config: %+v, err = %v", cfg, err)
	}

	t.Setenv("EMAIL_ENABLED", "ya")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "EMAIL_ENABLED") {
		t.Errorf("invalid bool: err = %v", err)
	}
}

func TestLoad_ProxyAndTurnstile(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	if err != nil || cfg.TrustedProxyHops != 0 || cfg.TurnstileSecret != "" {
		t.Fatalf("defaults: hops = %d, secret = %q, err = %v", cfg.TrustedProxyHops, cfg.TurnstileSecret, err)
	}

	t.Setenv("TRUSTED_PROXY_HOPS", "6")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXY_HOPS") {
		t.Errorf("hops out of range: err = %v", err)
	}
	t.Setenv("TRUSTED_PROXY_HOPS", "2")

	t.Setenv("APP_ENV", EnvProduction)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TURNSTILE_SECRET_KEY") {
		t.Errorf("production without turnstile secret: err = %v", err)
	}

	t.Setenv("TURNSTILE_SECRET_KEY", "1x0000000000000000000000000000000AA")
	cfg, err = Load()
	if err != nil || cfg.TrustedProxyHops != 2 {
		t.Errorf("production config: hops = %d, err = %v", cfg.TrustedProxyHops, err)
	}
}

func TestLoad_Cloudinary(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("CLOUDINARY_CLOUD_NAME", "lumiere-demo")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "CLOUDINARY") {
		t.Fatalf("partial config: err = %v", err)
	}

	t.Setenv("CLOUDINARY_API_KEY", "123456")
	t.Setenv("CLOUDINARY_API_SECRET", "rahasia")
	cfg, err := Load()
	if err != nil || cfg.CloudinaryCloud != "lumiere-demo" || cfg.CloudinarySecret != "rahasia" {
		t.Errorf("full config: %+v, err = %v", cfg, err)
	}
}
