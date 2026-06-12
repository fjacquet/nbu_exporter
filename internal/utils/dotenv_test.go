package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fjacquet/nbu_exporter/internal/models"
)

func TestLoadDotEnvSetsUnsetVars(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DOTENV_TEST_HOST=h1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTENV_TEST_HOST", "") // register cleanup, then unset for real
	_ = os.Unsetenv("DOTENV_TEST_HOST")

	LoadDotEnv(cfg)
	if got := os.Getenv("DOTENV_TEST_HOST"); got != "h1" {
		t.Errorf("DOTENV_TEST_HOST = %q, want h1", got)
	}
}

func TestLoadDotEnvNeverOverridesRealEnv(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DOTENV_TEST_PW=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTENV_TEST_PW", "from-env")

	LoadDotEnv(cfg)
	if got := os.Getenv("DOTENV_TEST_PW"); got != "from-env" {
		t.Errorf("DOTENV_TEST_PW = %q, want from-env (real env must win)", got)
	}
}

func TestLoadDotEnvMissingFileIsNoop(t *testing.T) {
	LoadDotEnv(filepath.Join(t.TempDir(), "config.yaml")) // must not panic or log fatal
}

func TestLoadDotEnvFeedsInterpolation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(filepath.Join(dir, ".env"),
		[]byte("NBU1_HOSTNAME=nbu.example.com\nNBU1_APIKEY=s3cret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(`
server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "1h"
  logName: "test.log"
nbuserver:
  scheme: "https"
  uri: "/netbackup"
  port: "1556"
  host: "${NBU1_HOSTNAME}"
  apiKey: "${NBU1_APIKEY}"
  apiVersion: "13.0"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"NBU1_HOSTNAME", "NBU1_APIKEY"} {
		t.Setenv(v, "")
		_ = os.Unsetenv(v)
	}

	LoadDotEnv(cfgPath)
	var cfg models.Config
	if err := ReadFile(&cfg, cfgPath); err != nil {
		t.Fatal(err)
	}
	if cfg.NbuServer.Host != "nbu.example.com" || cfg.NbuServer.APIKey != "s3cret" {
		t.Errorf("interpolated nbuserver = host:%q apiKey:%q", cfg.NbuServer.Host, cfg.NbuServer.APIKey)
	}
}
