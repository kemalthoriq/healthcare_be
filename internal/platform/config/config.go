package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	AppEnv          string
	Port            int
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	var errs []error

	cfg := Config{
		AppEnv: getEnv("APP_ENV", EnvDevelopment),
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

	return cfg, errors.Join(errs...)
}

func (c Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
