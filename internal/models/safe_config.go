// Package models defines the core data structures for the NBU exporter application.
package models

import (
	"fmt"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// SafeConfig provides thread-safe access to configuration.
// It uses RWMutex to allow concurrent reads while serializing writes.
// Pattern from Prometheus blackbox_exporter.
//
// SafeConfig enables dynamic configuration reload without restarting the exporter:
//   - Operators can update credentials or server addresses via SIGHUP
//   - File watchers can trigger automatic reload when config files change
//   - Invalid configurations are rejected without affecting the running config
//
// Usage:
//
//	cfg := &models.Config{...}
//	safeCfg := NewSafeConfig(cfg)
//
//	// Read (concurrent-safe)
//	current := safeCfg.Get()
//
//	// Reload (validates before applying)
//	changed, err := safeCfg.ReloadConfig("/path/to/config.yaml")
type SafeConfig struct {
	mu sync.RWMutex
	C  *Config
}

// NewSafeConfig creates a new SafeConfig with the provided initial config.
// The config is stored by reference; the caller should not modify it after
// passing it to NewSafeConfig.
func NewSafeConfig(cfg *Config) *SafeConfig {
	return &SafeConfig{
		C: cfg,
	}
}

// Get returns the current configuration (read-locked).
// The returned pointer is safe to use until the next reload.
// Multiple goroutines can call Get() concurrently.
func (sc *SafeConfig) Get() *Config {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.C
}

// ReloadConfig loads and validates a new configuration from the file.
// Validation happens BEFORE acquiring write lock (fail-fast pattern).
// This ensures invalid configurations never affect the running exporter.
//
// Returns:
//   - serverChanged: true if NBU server address changed (signals cache flush needed)
//   - err: error if file cannot be read or validation fails
//
// Thread-safety:
// The write lock is held only for the pointer swap operation, minimizing
// contention. Validation and file I/O happen without holding any locks.
func (sc *SafeConfig) ReloadConfig(configPath string) (serverChanged bool, err error) {
	// Step 1: Validate config WITHOUT holding any locks

	// Check file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read and parse config file
	f, err := os.Open(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to open config: %w", err)
	}
	defer func() { _ = f.Close() }()

	var newCfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&newCfg); err != nil {
		return false, fmt.Errorf("failed to decode config: %w", err)
	}

	// Validate config
	if err := newCfg.Validate(); err != nil {
		return false, fmt.Errorf("config validation failed: %w", err)
	}

	// Step 2: Acquire write lock ONLY for pointer swap
	sc.mu.Lock()
	oldHost := sc.C.NbuServer.Host
	oldPort := sc.C.NbuServer.Port
	sc.C = &newCfg
	sc.mu.Unlock()

	// Check if NBU server changed (signals cache flush needed)
	serverChanged = oldHost != newCfg.NbuServer.Host || oldPort != newCfg.NbuServer.Port

	log.Info("Configuration reloaded successfully")
	if serverChanged {
		log.Info("NBU server address changed, cache will be flushed")
	}

	return serverChanged, nil
}
