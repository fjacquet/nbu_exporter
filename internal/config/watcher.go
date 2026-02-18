// Package config provides configuration management utilities including
// file watching and signal handling for dynamic configuration reload.
package config

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

// ReloadFunc is called when config reload is triggered.
// Returns error if reload fails (logged but doesn't stop watcher).
// The configPath parameter is the path to the configuration file.
type ReloadFunc func(configPath string) error

// SetupSIGHUPHandler sets up SIGHUP signal handler for config reload.
// SIGHUP is the standard Unix signal for configuration reload.
// Runs in a goroutine, returns immediately.
//
// The handler:
//   - Listens for SIGHUP signals on a buffered channel
//   - Calls reloadFn when signal is received
//   - Logs success or failure of reload operation
//   - Continues listening after reload (reusable handler)
//
// Usage:
//
//	SetupSIGHUPHandler("/path/to/config.yaml", server.ReloadConfig)
//	// Now: kill -HUP <pid> triggers reload
func SetupSIGHUPHandler(configPath string, reloadFn ReloadFunc) {
	// Buffered channel prevents signal loss if handler is busy
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	go func() {
		for {
			<-sighup
			log.Info("SIGHUP received, reloading configuration...")
			if err := reloadFn(configPath); err != nil {
				log.Errorf("Configuration reload failed: %v", err)
			}
		}
	}()

	log.Info("SIGHUP handler configured for config reload")
}

// WatchConfigFile watches config file for changes and triggers reload.
//
// IMPORTANT: Watches directory (not file) for atomic write compatibility.
// Text editors like vim and emacs use atomic writes (write to temp file,
// then rename). Watching the file directly misses these changes because
// the original inode is replaced. Watching the directory catches both
// direct writes and atomic renames.
//
// The watcher:
//   - Watches the directory containing the config file
//   - Filters events to only react to the specific config file
//   - Triggers reload on Write or Create events (covers both edit and atomic save)
//   - Logs errors but continues watching (graceful degradation)
//
// Returns the watcher for cleanup (caller should defer watcher.Close()).
// Returns error if watcher creation or directory watch setup fails.
//
// Usage:
//
//	watcher, err := WatchConfigFile("/path/to/config.yaml", server.ReloadConfig)
//	if err != nil {
//	    log.Warnf("File watcher setup failed: %v", err)
//	} else {
//	    defer watcher.Close()
//	}
func WatchConfigFile(configPath string, reloadFn ReloadFunc) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch directory (not file) - editors use atomic writes (temp file + rename)
	// which would break file-level watching since the inode changes
	configDir := filepath.Dir(configPath)
	configName := filepath.Base(configPath)

	if err := watcher.Add(configDir); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					// Channel closed, watcher stopped
					return
				}
				// Filter for our config file and write/create events
				if filepath.Base(event.Name) == configName {
					if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
						log.Info("Config file changed, reloading...")
						if err := reloadFn(configPath); err != nil {
							log.Errorf("Configuration reload failed: %v", err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					// Channel closed, watcher stopped
					return
				}
				log.Errorf("File watcher error: %v", err)
			}
		}
	}()

	log.Infof("Watching config file: %s", configPath)
	return watcher, nil
}
