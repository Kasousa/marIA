// Package hello is the Phase 1 service: it proves the pipeline end to end
// with a single health endpoint.
package hello

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

// Config is the hello service configuration, read from environment variables.
type Config struct {
	Env      string     // ENV: staging or prod (required)
	Port     string     // PORT: HTTP port (default 8080)
	Version  string     // VERSION: image tag, shown by /healthz (default "dev")
	LogLevel slog.Level // LOG_LEVEL: debug, info, warn or error (default info)
}

// Port returns the HTTP port from PORT (default 8080). The healthcheck
// subcommand uses it, so it works even without the rest of the configuration.
func Port() string {
	return withDefault(os.Getenv("PORT"), "8080")
}

// Load reads the configuration from the process environment.
func Load() (Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Env:     getenv("ENV"),
		Port:    withDefault(getenv("PORT"), "8080"),
		Version: withDefault(getenv("VERSION"), "dev"),
	}

	var errs []error
	switch cfg.Env {
	case "staging", "prod":
	case "":
		errs = append(errs, errors.New("ENV is required"))
	default:
		errs = append(errs, fmt.Errorf("ENV must be staging or prod, got %q", cfg.Env))
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(withDefault(getenv("LOG_LEVEL"), "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}
	return cfg, errors.Join(errs...)
}

func withDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
