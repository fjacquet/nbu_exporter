// Package utils provides utility functions for file operations and configuration management.
package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envRefPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-[^}]*)?\}`)

// ExpandEnv replaces ${VAR} references with the value of the environment variable VAR.
// It returns an error if a referenced variable is not set, so misconfiguration fails
// loudly at startup rather than silently authenticating with an empty secret.
//
// A reference may carry a fallback as ${VAR:-default}, borrowing the shell /
// docker-compose syntax and its meaning: unset OR empty falls back, and the reference
// never errors. That lets a shipped config.yaml drive a non-secret setting from the
// environment while still starting on a host that never exported it. Use it only where a
// safe default exists — a bare ${VAR} keeps the fail-loud behaviour that protects secrets.
func ExpandEnv(s string) (string, error) {
	var missing []string
	out := envRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := envRefPattern.FindStringSubmatch(match)
		name, fallback := sub[1], sub[2]
		val, ok := os.LookupEnv(name)
		if ok && val != "" {
			return val
		}
		if fallback != "" {
			return fallback[len(":-"):] // group 2 keeps its ":-" prefix, so "" means absent
		}
		if !ok {
			missing = append(missing, name)
		}
		return ""
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("environment variable(s) referenced in config but not set: %s", strings.Join(missing, ", "))
	}
	return out, nil
}
