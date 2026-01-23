package models

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewSafeConfig(t *testing.T) {
	cfg := &Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = "2112"

	sc := NewSafeConfig(cfg)

	if sc == nil {
		t.Fatal("NewSafeConfig returned nil")
	}
	if sc.C != cfg {
		t.Error("SafeConfig.C does not point to the original config")
	}
}

func TestSafeConfigGet(t *testing.T) {
	cfg := &Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = "2112"
	cfg.NbuServer.Host = "nbu1.example.com"
	cfg.NbuServer.Port = "1556"

	sc := NewSafeConfig(cfg)

	got := sc.Get()
	if got.Server.Host != "localhost" {
		t.Errorf("Expected Server.Host=localhost, got %s", got.Server.Host)
	}
	if got.NbuServer.Host != "nbu1.example.com" {
		t.Errorf("Expected NbuServer.Host=nbu1.example.com, got %s", got.NbuServer.Host)
	}
}

func TestSafeConfigConcurrentAccess(t *testing.T) {
	cfg := &Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = "2112"
	cfg.NbuServer.Host = "nbu1.example.com"
	cfg.NbuServer.Port = "1556"
	sc := NewSafeConfig(cfg)

	var wg sync.WaitGroup
	// 100 concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := sc.Get()
			// Access config fields to ensure no race
			_ = got.Server.Host
			_ = got.NbuServer.Host
		}()
	}
	wg.Wait()
}

func TestSafeConfigReloadServerChanged(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
nbuserver:
  host: "nbu1.example.com"
  port: "1556"
  scheme: "https"
  apiKey: "test-api-key-12345678"
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	cfg := &Config{}
	cfg.NbuServer.Host = "nbu1.example.com"
	cfg.NbuServer.Port = "1556"
	sc := NewSafeConfig(cfg)

	// Reload with same server - should return false for serverChanged
	changed, err := sc.ReloadConfig(configPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if changed {
		t.Error("Expected serverChanged=false for same server")
	}

	// Update config with different server
	newConfig := `server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
nbuserver:
  host: "nbu2.example.com"
  port: "1556"
  scheme: "https"
  apiKey: "test-api-key-12345678"
`
	if err := os.WriteFile(configPath, []byte(newConfig), 0644); err != nil {
		t.Fatalf("Failed to write new config: %v", err)
	}

	changed, err = sc.ReloadConfig(configPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if !changed {
		t.Error("Expected serverChanged=true for different server")
	}

	// Verify the new config is applied
	got := sc.Get()
	if got.NbuServer.Host != "nbu2.example.com" {
		t.Errorf("Expected new host nbu2.example.com, got %s", got.NbuServer.Host)
	}
}

func TestSafeConfigReloadPortChanged(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
nbuserver:
  host: "nbu1.example.com"
  port: "1556"
  scheme: "https"
  apiKey: "test-api-key-12345678"
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	cfg := &Config{}
	cfg.NbuServer.Host = "nbu1.example.com"
	cfg.NbuServer.Port = "1556"
	sc := NewSafeConfig(cfg)

	// Change port only
	newConfig := `server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
nbuserver:
  host: "nbu1.example.com"
  port: "1557"
  scheme: "https"
  apiKey: "test-api-key-12345678"
`
	if err := os.WriteFile(configPath, []byte(newConfig), 0644); err != nil {
		t.Fatalf("Failed to write new config: %v", err)
	}

	changed, err := sc.ReloadConfig(configPath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if !changed {
		t.Error("Expected serverChanged=true when port changes")
	}
}

func TestSafeConfigReloadFileNotFound(t *testing.T) {
	cfg := &Config{}
	cfg.NbuServer.Host = "nbu1.example.com"
	cfg.NbuServer.Port = "1556"
	sc := NewSafeConfig(cfg)

	_, err := sc.ReloadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestSafeConfigReloadInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid config (missing required fields)
	invalidConfig := `server:
  host: ""
  port: ""
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Start with valid config
	cfg := &Config{}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = "2112"
	cfg.Server.URI = "/metrics"
	cfg.Server.ScrapingInterval = "5m"
	cfg.NbuServer.Host = "nbu1.example.com"
	cfg.NbuServer.Port = "1556"
	cfg.NbuServer.Scheme = "https"
	cfg.NbuServer.APIKey = "test-key"
	sc := NewSafeConfig(cfg)

	// Reload with invalid config should fail
	_, err := sc.ReloadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid config")
	}

	// Original config should be preserved
	got := sc.Get()
	if got.Server.Host != "localhost" {
		t.Errorf("Expected original host preserved, got %s", got.Server.Host)
	}
}

func TestSafeConfigReloadMalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write malformed YAML
	malformedConfig := `server:
  host: "localhost
  port: 2112
    invalid indent
`
	if err := os.WriteFile(configPath, []byte(malformedConfig), 0644); err != nil {
		t.Fatalf("Failed to write malformed config: %v", err)
	}

	cfg := &Config{}
	cfg.Server.Host = "original"
	sc := NewSafeConfig(cfg)

	_, err := sc.ReloadConfig(configPath)
	if err == nil {
		t.Error("Expected error for malformed YAML")
	}

	// Original config should be preserved
	got := sc.Get()
	if got.Server.Host != "original" {
		t.Errorf("Expected original host preserved, got %s", got.Server.Host)
	}
}

func TestSafeConfigConcurrentReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	validConfig := `server:
  host: "localhost"
  port: "2112"
  uri: "/metrics"
  scrapingInterval: "5m"
nbuserver:
  host: "nbu1.example.com"
  port: "1556"
  scheme: "https"
  apiKey: "test-api-key-12345678"
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg := &Config{}
	cfg.NbuServer.Host = "nbu1.example.com"
	cfg.NbuServer.Port = "1556"
	sc := NewSafeConfig(cfg)

	var wg sync.WaitGroup

	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				got := sc.Get()
				_ = got.Server.Host
			}
		}()
	}

	// Concurrent reloaders
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				_, _ = sc.ReloadConfig(configPath)
			}
		}()
	}

	wg.Wait()
}
