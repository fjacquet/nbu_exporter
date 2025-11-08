// Package utils provides utility functions for file operations and configuration management.
package utils

import (
	"fmt"
	"os"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"gopkg.in/yaml.v2"
)

// FileExists checks if the given file exists and is accessible.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// ReadFile reads and parses a YAML configuration file into the provided Config struct.
//
// Parameters:
//   - cfg: Pointer to Config struct to populate
//   - filepath: Path to the YAML configuration file
//
// Returns an error if:
//   - The file cannot be opened
//   - The YAML cannot be parsed
//   - The structure doesn't match the Config model
//
// Example:
//
//	var cfg models.Config
//	err := ReadFile(&cfg, "config.yaml")
func ReadFile(cfg *models.Config, filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open config file %s: %w", filepath, err)
	}
	defer func() {
		_ = f.Close()
	}()

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to decode config file %s: %w", filepath, err)
	}

	return nil
}
