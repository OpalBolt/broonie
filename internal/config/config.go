package config

import (
	"crypto/sha256"
	"errors"
	"os"
	"time"
)

// Config holds the application configuration.
type Config struct {
	SecretKey     string
	SessionDir    string
	DBPath        string
	PollInterval  time.Duration
	EncryptionKey [32]byte
}

// LoadConfig reads configuration from environment variables.
// Defaults: DBPath="./broonie.db", PollInterval=60s.
// Returns error if SECRET_KEY is not set.
func LoadConfig() (Config, error) {
	cfg := Config{
		DBPath:       "./broonie.db",
		PollInterval: 60 * time.Second,
	}

	cfg.SecretKey = os.Getenv("SECRET_KEY")
	if cfg.SecretKey == "" {
		return cfg, errors.New("SECRET_KEY environment variable is required")
	}

	cfg.SessionDir = os.Getenv("PI_CODING_AGENT_SESSION_DIR")

	if path := os.Getenv("DB_PATH"); path != "" {
		cfg.DBPath = path
	}

	cfg.EncryptionKey = sha256.Sum256([]byte(cfg.SecretKey))
	return cfg, nil
}
