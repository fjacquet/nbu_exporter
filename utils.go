package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

// test if a file exists
//
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

// readFile do read a yaml file
func ReadFile(Cfg *Config, filepath string) {
	f, err := os.Open(filepath)
	if err != nil {
		ProcessError(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(Cfg)
	if err != nil {
		ProcessError(err)
	}
}
