package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	if err := os.WriteFile(configPath, []byte("initial: content"), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	var reloadCount int32
	reloadFn := func(path string) error {
		atomic.AddInt32(&reloadCount, 1)
		return nil
	}

	watcher, err := WatchConfigFile(configPath, reloadFn)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	if err := os.WriteFile(configPath, []byte("updated: content"), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Wait for reload - use polling with timeout
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&reloadCount) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if atomic.LoadInt32(&reloadCount) == 0 {
		t.Error("Expected reload to be triggered")
	}
}

func TestWatchConfigFileAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	if err := os.WriteFile(configPath, []byte("initial: content"), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	var reloadCount int32
	reloadFn := func(path string) error {
		atomic.AddInt32(&reloadCount, 1)
		return nil
	}

	watcher, err := WatchConfigFile(configPath, reloadFn)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Simulate atomic write (like vim does): write to temp, then rename
	tempPath := filepath.Join(tmpDir, "config.yaml.tmp")
	if err := os.WriteFile(tempPath, []byte("atomic: content"), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := os.Rename(tempPath, configPath); err != nil {
		t.Fatalf("Failed to rename temp file: %v", err)
	}

	// Wait for reload
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&reloadCount) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if atomic.LoadInt32(&reloadCount) == 0 {
		t.Error("Expected reload to be triggered on atomic write")
	}
}

func TestWatchConfigFileReloadError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	if err := os.WriteFile(configPath, []byte("initial: content"), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	var reloadCount int32
	reloadFn := func(path string) error {
		atomic.AddInt32(&reloadCount, 1)
		return errors.New("reload failed")
	}

	watcher, err := WatchConfigFile(configPath, reloadFn)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	if err := os.WriteFile(configPath, []byte("updated: content"), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Wait for reload attempt
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&reloadCount) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Reload should have been attempted even though it failed
	if atomic.LoadInt32(&reloadCount) == 0 {
		t.Error("Expected reload to be attempted despite error")
	}
}

func TestWatchConfigFileOtherFileIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	otherPath := filepath.Join(tmpDir, "other.yaml")

	// Create initial config
	if err := os.WriteFile(configPath, []byte("initial: content"), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	var reloadCount int32
	reloadFn := func(path string) error {
		atomic.AddInt32(&reloadCount, 1)
		return nil
	}

	watcher, err := WatchConfigFile(configPath, reloadFn)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify a different file in the same directory
	if err := os.WriteFile(otherPath, []byte("other: content"), 0644); err != nil {
		t.Fatalf("Failed to write other file: %v", err)
	}

	// Wait a bit to ensure no reload triggered
	time.Sleep(300 * time.Millisecond)

	if atomic.LoadInt32(&reloadCount) != 0 {
		t.Error("Expected no reload for changes to other files")
	}
}

func TestWatchConfigFileNonexistentDir(t *testing.T) {
	configPath := "/nonexistent/path/config.yaml"

	reloadFn := func(path string) error {
		return nil
	}

	_, err := WatchConfigFile(configPath, reloadFn)
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestWatchConfigFileClose(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	if err := os.WriteFile(configPath, []byte("initial: content"), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	var reloadCount int32
	reloadFn := func(path string) error {
		atomic.AddInt32(&reloadCount, 1)
		return nil
	}

	watcher, err := WatchConfigFile(configPath, reloadFn)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Close the watcher
	watcher.Close()

	// Wait for goroutine to exit
	time.Sleep(100 * time.Millisecond)

	// Modify config file after close
	if err := os.WriteFile(configPath, []byte("updated: content"), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// No reload should happen after close
	if atomic.LoadInt32(&reloadCount) != 0 {
		t.Error("Expected no reload after watcher closed")
	}
}
