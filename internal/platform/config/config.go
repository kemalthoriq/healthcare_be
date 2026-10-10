package config

import (
	"errors"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	AppEnv            string
	Port              int
	ShutdownTimeout   time.Duration
	DatabaseURL       string
	DBMaxConns        int32
	SeedAdminName     string
	SeedAdminEmail    string
	SeedAdminPassword string
	FrontendOrigin    string
	SessionTTL        time.Duration
}

func Load() (Config, error) {
	var errs []error

	cfg := Config{
		AppEnv:            getEnv("APP_ENV", EnvDevelopment),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		SeedAdminName:     getEnv("SEED_ADMIN_NAME", "Admin"),
		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", ""),
		SeedAdminPassword: getEnv("SEED_ADMIN_PASSWORD", ""),
		FrontendOrigin:    getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
	}
	if cfg.AppEnv != EnvDevelopment && cfg.AppEnv != EnvProduction {
		errs = append(errs, errors.New("APP_ENV must be either development or production"))
	}

	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil || port < 1 || port > 65535 {
		errs = append(errs, errors.New("PORT must be a number between 1 and 65535"))
	}
	cfg.Port = port

	timeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil || timeout <= 0 {
		errs = append(errs, errors.New("SHUTDOWN_TIMEOUT must be a positive duration, e.g. 10s"))
	}
	cfg.ShutdownTimeout = timeout

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}

	maxConns, err := strconv.ParseInt(getEnv("DB_MAX_CONNS", "5"), 10, 32)
	if err != nil || maxConns < 1 || maxConns > 50 {
		errs = append(errs, errors.New("DB_MAX_CONNS must be a number between 1 and 50"))
	}
	cfg.DBMaxConns = int32(maxConns)

	if cfg.SeedAdminEmail != "" || cfg.SeedAdminPassword != "" {
		if _, err := mail.ParseAddress(cfg.SeedAdminEmail); err != nil {
			errs = append(errs, errors.New("SEED_ADMIN_EMAIL must be a valid email address"))
		}
		if len(cfg.SeedAdminPassword) < 12 {
			errs = append(errs, errors.New("SEED_ADMIN_PASSWORD must be at least 12 characters"))
		}
	}

	if !validOrigin(cfg.FrontendOrigin) {
		errs = append(errs, errors.New("FRONTEND_ORIGIN must be an origin like https://example.com, without a path"))
	}

	sessionTTL, err := time.ParseDuration(getEnv("SESSION_TTL", "168h"))
	if err != nil || sessionTTL <= 0 {
		errs = append(errs, errors.New("SESSION_TTL must be a positive duration, e.g. 168h"))
	}
	cfg.SessionTTL = sessionTTL

	return cfg, errors.Join(errs...)
}

func (c Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

func validOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	return u.Path == "" && u.RawQuery == "" && u.Fragment == "" && u.User == nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
