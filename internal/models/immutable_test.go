package models

import (
	"testing"
	"time"
)

func createValidConfig() *Config {
	cfg := &Config{}
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = "2112"
	cfg.Server.URI = "/metrics"
	cfg.Server.ScrapingInterval = "5m"
	cfg.Server.LogName = "test.log"

	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.Host = "nbu.example.com"
	cfg.NbuServer.Port = "1556"
	cfg.NbuServer.URI = "/netbackup"
	cfg.NbuServer.APIKey = "test-api-key-12345678"
	cfg.NbuServer.APIVersion = "13.0"
	cfg.NbuServer.InsecureSkipVerify = false

	cfg.OpenTelemetry.Enabled = true
	cfg.OpenTelemetry.Endpoint = "localhost:4317"
	cfg.OpenTelemetry.Insecure = true
	cfg.OpenTelemetry.SamplingRate = 0.5

	return cfg
}

func TestNewImmutableConfig_Success(t *testing.T) {
	cfg := createValidConfig()

	immutable, err := NewImmutableConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify NBU settings
	expectedURL := "https://nbu.example.com:1556/netbackup"
	if immutable.BaseURL() != expectedURL {
		t.Errorf("expected baseURL %s, got %s", expectedURL, immutable.BaseURL())
	}
	if immutable.APIKey() != "test-api-key-12345678" {
		t.Error("API key mismatch")
	}
	if immutable.APIVersion() != "13.0" {
		t.Errorf("expected API version 13.0, got %s", immutable.APIVersion())
	}
	if immutable.InsecureSkipVerify() != false {
		t.Error("expected InsecureSkipVerify false")
	}

	// Verify server settings
	if immutable.ServerAddress() != "0.0.0.0:2112" {
		t.Errorf("expected server address 0.0.0.0:2112, got %s", immutable.ServerAddress())
	}
	if immutable.MetricsURI() != "/metrics" {
		t.Errorf("expected metrics URI /metrics, got %s", immutable.MetricsURI())
	}
	if immutable.ScrapingInterval() != 5*time.Minute {
		t.Errorf("expected 5m, got %v", immutable.ScrapingInterval())
	}

	// Verify OTel settings
	if !immutable.OTelEnabled() {
		t.Error("expected OTel enabled")
	}
	if immutable.OTelSamplingRate() != 0.5 {
		t.Errorf("expected sampling rate 0.5, got %f", immutable.OTelSamplingRate())
	}
}

func TestNewImmutableConfig_InvalidScrapingInterval(t *testing.T) {
	cfg := createValidConfig()
	cfg.Server.ScrapingInterval = "invalid"

	_, err := NewImmutableConfig(cfg)
	if err == nil {
		t.Error("expected error for invalid scraping interval")
	}
}

func TestImmutableConfig_MaskedAPIKey(t *testing.T) {
	cfg := createValidConfig()
	immutable, _ := NewImmutableConfig(cfg)

	masked := immutable.MaskedAPIKey()
	if masked != "test****5678" {
		t.Errorf("expected test****5678, got %s", masked)
	}
}

func TestImmutableConfig_MaskedAPIKey_Short(t *testing.T) {
	cfg := createValidConfig()
	cfg.NbuServer.APIKey = "short"
	immutable, _ := NewImmutableConfig(cfg)

	masked := immutable.MaskedAPIKey()
	if masked != "****" {
		t.Errorf("expected ****, got %s", masked)
	}
}

func TestImmutableConfig_ScrapingIntervalString(t *testing.T) {
	cfg := createValidConfig()
	immutable, _ := NewImmutableConfig(cfg)

	intervalStr := immutable.ScrapingIntervalString()
	if intervalStr != "5m0s" {
		t.Errorf("expected 5m0s, got %s", intervalStr)
	}
}

func TestImmutableConfig_ValuesAreSnapshots(t *testing.T) {
	cfg := createValidConfig()
	immutable, _ := NewImmutableConfig(cfg)

	// Modify original config
	cfg.NbuServer.APIVersion = "99.0"
	cfg.NbuServer.Host = "modified.example.com"

	// ImmutableConfig should retain original values
	if immutable.APIVersion() != "13.0" {
		t.Error("ImmutableConfig should not be affected by Config changes")
	}
	if immutable.BaseURL() != "https://nbu.example.com:1556/netbackup" {
		t.Error("ImmutableConfig baseURL should not be affected by Config changes")
	}
}
