// Package utils provides utility functions for file operations and configuration management.
package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envRefPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ExpandEnv replaces ${VAR} references with the value of the environment variable VAR.
// It returns an error if a referenced variable is not set, so misconfiguration fails
// loudly at startup rather than silently authenticating with an empty secret.
func ExpandEnv(s string) (string, error) {
	var missing []string
	out := envRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := envRefPattern.FindStringSubmatch(match)[1]
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return ""
		}
		return val
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("environment variable(s) referenced in config but not set: %s", strings.Join(missing, ", "))
	}
	return out, nil
}
