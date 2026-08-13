package utils

import (
	"os"
	"testing"
)

// TestExpandFallback pins the ${VAR:-default} form ported from pscale_exporter: it falls
// back when the variable is unset OR exported empty (shell / docker-compose semantics),
// prefers a real value, and never errors — while a bare ${VAR} must keep failing loudly,
// which is what stops an UNSET variable from silently resolving to an empty string.
func TestExpandFallback(t *testing.T) {
	unsetForTest(t, "NBU_FALLBACK_TEST_UNSET")
	t.Setenv("NBU_FALLBACK_TEST_SET", "real")
	t.Setenv("NBU_FALLBACK_TEST_EMPTY", "")
	for _, tc := range []struct{ name, in, want string }{
		{"unset falls back", "${NBU_FALLBACK_TEST_UNSET:-false}", "false"},
		{"set wins over default", "${NBU_FALLBACK_TEST_SET:-false}", "real"},
		{"exported empty falls back", "${NBU_FALLBACK_TEST_EMPTY:-other}", "other"},
		{"empty default allowed", "${NBU_FALLBACK_TEST_UNSET:-}", ""},
		{"mixed with literal text", "a${NBU_FALLBACK_TEST_UNSET:-b}c", "abc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandEnv(tc.in)
			if err != nil {
				t.Fatalf("ExpandEnv(%q): unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
	if _, err := ExpandEnv("${NBU_FALLBACK_TEST_UNSET}"); err == nil {
		t.Error("a bare reference to an unset variable must still fail")
	}
}

// unsetForTest clears name for the duration of the test and restores whatever was there —
// value and set/unset state alike. Tests that assert on an *unset* variable are otherwise
// at the mercy of whatever the developer or CI runner happens to export.
func unsetForTest(t *testing.T, name string) {
	t.Helper()
	old, had := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("unset %s: %v", name, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(name, old)
			return
		}
		_ = os.Unsetenv(name)
	})
}

// TestExpandSecretRejectsEmpty pins the credential-only strictness: plain expansion lets an
// exported-but-empty variable through (matching os.Expand and long-standing behaviour), but
// a credential written as a reference that resolves to nothing is a misconfiguration —
// without this the exporter would authenticate with an empty credential and blame the
// appliance.
func TestExpandSecretRejectsEmpty(t *testing.T) {
	t.Setenv("NBU_EMPTY_SECRET_TEST", "")

	if _, err := ExpandEnv("${NBU_EMPTY_SECRET_TEST}"); err != nil {
		t.Fatalf("plain expansion must stay lenient on an exported-empty variable: %v", err)
	}
	if _, err := ExpandEnvSecret("password", "${NBU_EMPTY_SECRET_TEST}"); err == nil {
		t.Error("a credential resolving to an empty value must be rejected")
	}
	// A literal credential is not a reference and must pass through untouched, as must an
	// omitted optional one — otherwise passwordFile setups would break.
	if got, err := ExpandEnvSecret("password", "literal-pw"); err != nil || got != "literal-pw" {
		t.Errorf("literal credential: got %q err=%v", got, err)
	}
	if got, err := ExpandEnvSecret("password", ""); err != nil || got != "" {
		t.Errorf("omitted credential: got %q err=%v", got, err)
	}
}
