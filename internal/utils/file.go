package utils

import (
	"os"

	"github.com/fjacquet/nbu_exporter/internal/logging"
	"github.com/fjacquet/nbu_exporter/internal/models"
	"gopkg.in/yaml.v2"
)

// test if a file exists
//
// fileExists checks if the given file exists.
// It returns true if the file exists, and false otherwise.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// readFile do read a yaml file
// ReadFile reads the configuration from the specified YAML file.
//
// It opens the file, creates a YAML decoder, and decodes the configuration into the provided Config struct.
// If any errors occur during the process, they are passed to the ProcessError function.
func ReadFile(Cfg *models.Config, filepath string) {
	f, err := os.Open(filepath)
	if err != nil {
		logging.HandleError(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(Cfg)
	if err != nil {
		logging.HandleError(err)
		return
	}
}
