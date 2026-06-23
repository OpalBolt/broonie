package config

import (
	"crypto/sha256"
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration.
type Config struct {
	SecretKey     string
	SessionDir    string
	DBPath        string
	PollInterval  time.Duration
	EncryptionKey [32]byte
	MaxIterations int    // max retry loops for agent (default 3)
	NonoProfile   string // path to nono profile JSON
	ModelsPath    string // path to .pi/models.json for MODEL_ patching
}

// LoadConfig reads configuration from environment variables.
// Defaults: DBPath="./broonie.db", PollInterval=60s.
// Returns error if SECRET_KEY is not set.
func LoadConfig() (Config, error) {
	cfg := Config{
		DBPath:       "./broonie.db",
		PollInterval: 60 * time.Second,
		MaxIterations: 3,
		NonoProfile:   "nono/broonie-go.json",
		ModelsPath:    ".pi/models.json",
	}

	cfg.SecretKey = os.Getenv("SECRET_KEY")
	if cfg.SecretKey == "" {
		return cfg, errors.New("SECRET_KEY environment variable is required")
	}

	cfg.SessionDir = os.Getenv("PI_CODING_AGENT_SESSION_DIR")

	if path := os.Getenv("DB_PATH"); path != "" {
		cfg.DBPath = path
	}

	if maxIter := os.Getenv("MAX_ITERATIONS"); maxIter != "" {
		// ponytail: simple atoi, error treated as 0, falls back to default 3
		if n, err := strconv.Atoi(maxIter); err == nil && n > 0 {
			cfg.MaxIterations = n
		}
	}

	if profile := os.Getenv("NONO_PROFILE"); profile != "" {
		cfg.NonoProfile = profile
	}

	if models := os.Getenv("MODELS_PATH"); models != "" {
		cfg.ModelsPath = models
	}
	cfg.EncryptionKey = sha256.Sum256([]byte(cfg.SecretKey))
	return cfg, nil
}
