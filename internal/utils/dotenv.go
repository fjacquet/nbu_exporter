// Package utils provides utility functions for file operations and configuration management.
package utils

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

// LoadDotEnv loads a .env file before config interpolation so the
// `cp .env.example .env` quickstart works for bare-metal runs too, not just
// docker compose (which reads .env natively). It tries the working directory
// first, then the config file's directory (covers systemd units whose
// WorkingDirectory is not the install dir). The first file found wins.
//
// Already-set environment variables always take precedence: godotenv.Load only
// sets keys that are not present in the environment, so real env/secret
// injection can never be shadowed by a stray .env file.
func LoadDotEnv(cfgPath string) {
	for _, p := range []string{".env", filepath.Join(filepath.Dir(cfgPath), ".env")} {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := godotenv.Load(p); err != nil {
			log.WithError(err).WithField("file", p).Warn("failed to load .env file")
			continue
		}
		log.WithField("file", p).Info("loaded .env (already-set environment variables take precedence)")
		return
	}
}
