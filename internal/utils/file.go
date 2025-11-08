package utils

import (
	"fmt"
	"os"

	"github.com/fjacquet/nbu_exporter/internal/models"
	"gopkg.in/yaml.v2"
)

// FileExists checks if the given file exists.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// ReadFile reads the configuration from the specified YAML file.
// It returns an error if the file cannot be opened or parsed.
func ReadFile(cfg *models.Config, filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open config file %s: %w", filepath, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to decode config file %s: %w", filepath, err)
	}

	return nil
}
