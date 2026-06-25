// Package config loads runtime configuration from the environment.
//
// Twelve-factor style: every knob is an env var, with sane local defaults so
// `go run ./cmd/server` works out of the box. A .env file (gitignored) is loaded
// first if present, then real environment variables win.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config is the fully-resolved application configuration.
type Config struct {
	Env  string // "dev" | "prod"
	Port string // HTTP listen port, e.g. "8080"

	DatabaseURL string // postgres connection string

	// AnthropicAPIKey powers the optional "Get a Claude review" button on the
	// code checker. Empty = the review feature is disabled gracefully.
	AnthropicAPIKey string

	// CheckerTimeoutSec bounds how long a single `go test` run may take.
	CheckerTimeoutSec int

	// BasicAuthUser/Password gate the whole app behind HTTP Basic Auth when BOTH
	// are set. Strongly recommended for any public deployment, since the code
	// checker executes `go test` on the server — you don't want strangers running
	// code on it. Empty = no auth (fine for local single-user use).
	BasicAuthUser     string
	BasicAuthPassword string
}

// Load reads configuration from the environment (and an optional .env file).
func Load() (*Config, error) {
	// Best-effort: a missing .env is fine in production.
	_ = godotenv.Load()

	cfg := &Config{
		Env:               getenv("APP_ENV", "dev"),
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       getenv("DATABASE_URL", defaultDatabaseURL()),
		AnthropicAPIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		CheckerTimeoutSec: getenvInt("CHECKER_TIMEOUT_SEC", 45),
		BasicAuthUser:     os.Getenv("APP_USER"),
		BasicAuthPassword: os.Getenv("APP_PASSWORD"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

// IsProd reports whether we're running in the production environment.
func (c *Config) IsProd() bool { return c.Env == "prod" }

// defaultDatabaseURL points at a local Postgres named "consensus".
// On macOS Homebrew, the default superuser is your OS username.
func defaultDatabaseURL() string {
	user := os.Getenv("USER")
	if user == "" {
		user = "postgres"
	}
	return fmt.Sprintf("postgres://%s@localhost:5432/consensus?sslmode=disable", user)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}
