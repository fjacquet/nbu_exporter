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
// safe default exists.
//
// A bare ${VAR} fails when the variable is UNSET; an exported-but-empty one expands to
// the empty string, as it always has. Credential fields get the stricter treatment —
// see ExpandEnvSecret.
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

// ExpandEnvSecret expands like ExpandEnv, but additionally rejects a credential that was
// written as an env reference yet resolves to nothing. A stray `NBU1_PASSWORD=` line in
// a .env file is a plausible typo, and without this the exporter would authenticate with an
// empty credential and report a failure that names the wrong cause.
//
// It fires only when the field actually contains a ${...} reference: a literal value is
// passed through untouched and an omitted optional credential stays omitted, so it cannot
// break a config that never referenced the environment in the first place.
func ExpandEnvSecret(field, s string) (string, error) {
	out, err := ExpandEnv(s)
	if err != nil {
		return "", err
	}
	// The message names the FIELD and nothing else. Config-load errors are logged, and
	// every part of s is potentially secret — even the variable name is a substring of a
	// credential field, which is why it is not echoed either. The offending reference is
	// visible in config.yaml next to the field this names.
	if out == "" && envRefPattern.MatchString(s) {
		return "", fmt.Errorf("%s is set from an environment reference that resolved to an "+
			"empty value; export the variable or remove the reference", field)
	}
	return out, nil
}
