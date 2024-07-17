package main

import (
	"os"
	"testing"
)

// TestCheckParams tests the checkParams function in various scenarios.
// It checks that the program exits with an error when no arguments are provided,
// and that it does not exit when valid arguments are provided or when a config
// file exists. It also tests the case where the config file does not exist.
// func TestCheckParams(t *testing.T) {
// 	// Test case: No arguments provided
// 	os.Args = []string{"testprog"}
// 	defer func() {
// 		if r := recover(); r == nil {
// 			t.Errorf("Expected program to exit with error, but it didn't")
// 		}
// 	}()
// 	checkParams()

// 	// Test case: Valid argument provided
// 	os.Args = []string{"testprog", "--help"}
// 	defer func() {
// 		if r := recover(); r != nil {
// 			t.Errorf("Expected program to not exit, but it did: %v", r)
// 		}
// 	}()
// 	checkParams()

// 	// Test case: Config file exists
// 	ConfigFile = "testfile.txt"
// 	defer func() {
// 		os.Remove(ConfigFile)
// 		if r := recover(); r != nil {
// 			t.Errorf("Expected program to not exit, but it did: %v", r)
// 		}
// 	}()
// 	file, err := os.Create(ConfigFile)
// 	if err != nil {
// 		t.Errorf("Failed to create test file: %v", err)
// 	}
// 	file.Close()
// 	checkParams()

// 	// Test case: Config file doesn't exist
// 	ConfigFile = "nonexistent.txt"
// 	defer func() {
// 		if r := recover(); r == nil {
// 			t.Errorf("Expected program to exit with error, but it didn't")
// 		}
// 	}()
// 	checkParams()
// }

// TestPrepareLogs tests the prepareLogs function by capturing the output
// and verifying that the log file is created and the expected output is
// written to both stdout and the log file.
// func TestPrepareLogs(t *testing.T) {
// 	// Save the original stdout and restore it after the test
// 	origStdout := os.Stdout
// 	defer func() {
// 		os.Stdout = origStdout
// 	}()

// 	// Create a buffer to capture the output
// 	var buf bytes.Buffer
// 	os.Stdout = &buf

// 	// Set a temporary log file name
// 	Cfg.Server.LogName = "test.log"
// 	defer os.Remove(Cfg.Server.LogName)

// 	prepareLogs()

// 	// Check if the log file was created
// 	_, err := os.Stat(Cfg.Server.LogName)
// 	if err != nil {
// 		t.Errorf("Failed to create log file: %v", err)
// 	}

// 	// Check if the output was written to both stdout and the log file
// 	expectedOutput := "Test output"
// 	log.Print(expectedOutput)
// 	if !bytes.Contains(buf.Bytes(), []byte(expectedOutput)) {
// 		t.Errorf("Output not written to stdout")
// 	}

// 	logData, err := os.ReadFile(Cfg.Server.LogName)
// 	if err != nil {
// 		t.Errorf("Failed to read log file: %v", err)
// 	}
// 	if !bytes.Contains(logData, []byte(expectedOutput)) {
// 		t.Errorf("Output not written to log file")
// 	}
// }

// TestPrepareLogsWithInvalidLogName tests the behavior of the prepareLogs function when
// the configured log file name is invalid. It saves the original stdout, sets an invalid
// log file name, and expects a panic to occur when prepareLogs is called.
func TestPrepareLogsWithInvalidLogName(t *testing.T) {
	// Save the original stdout and restore it after the test
	origStdout := os.Stdout
	defer func() {
		os.Stdout = origStdout
	}()

	// Set an invalid log file name
	Cfg.Server.LogName = "/invalid/path/test.log"
	defer func() {
		Cfg.Server.LogName = "test.log"
	}()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic, but didn't occur")
		}
	}()

	prepareLogs()
}

// TestMainWithInvalidArgs tests the main function with invalid command-line arguments.
// It saves the original os.Args, sets invalid arguments, and verifies that the program
// exits with an error.
// func TestMainWithInvalidArgs(t *testing.T) {
// 	// Save the original os.Args and restore it after the test
// 	origArgs := os.Args
// 	defer func() {
// 		os.Args = origArgs
// 	}()

// 	// Set invalid arguments
// 	os.Args = []string{"testprog", "--invalid-arg"}

// 	defer func() {
// 		if r := recover(); r == nil {
// 			t.Errorf("Expected program to exit with error, but it didn't")
// 		}
// 	}()

// 	main()
// }

// TestMainWithValidArgs tests the main function with valid command-line arguments.
// It saves the original os.Args, sets valid arguments, and verifies that the program
// does not exit with an error.
// func TestMainWithValidArgs(t *testing.T) {
// 	// Save the original os.Args and restore it after the test
// 	origArgs := os.Args
// 	defer func() {
// 		os.Args = origArgs
// 	}()

// 	// Set valid arguments
// 	os.Args = []string{"testprog", "--help"}

// 	defer func() {
// 		if r := recover(); r != nil {
// 			t.Errorf("Expected program to not exit, but it did: %v", r)
// 		}
// 	}()

// 	main()
// }

// TestMainWithConfigFileError tests the behavior of the main function when the
// configuration file is not found.
//
// It saves the original ConfigFile value, sets it to a non-existent file,
// and then calls the main function. It expects the program to exit with an
// error, otherwise it fails the test.
// func TestMainWithConfigFileError(t *testing.T) {
// 	// Save the original ConfigFile and restore it after the test
// 	origConfigFile := ConfigFile
// 	defer func() {
// 		ConfigFile = origConfigFile
// 	}()

// 	// Set a non-existent config file
// 	ConfigFile = "nonexistent.cfg"

// 	defer func() {
// 		if r := recover(); r == nil {
// 			t.Errorf("Expected program to exit with error, but it didn't")
// 		}
// 	}()

// 	main()
// }

// TestMainWithLogFileError tests the behavior of the main function when the
// log file specified in the configuration is not writable.
//
// It saves the original log file name, sets it to an invalid path, and then
// calls the main function. It expects the program to exit with an error,
// otherwise it fails the test.
// func TestMainWithLogFileError(t *testing.T) {
// 	// Save the original Cfg.Server.LogName and restore it after the test
// 	origLogName := Cfg.Server.LogName
// 	defer func() {
// 		Cfg.Server.LogName = origLogName
// 	}()

// 	// Set an invalid log file name
// 	Cfg.Server.LogName = "/invalid/path/test.log"

// 	defer func() {
// 		if r := recover(); r == nil {
// 			t.Errorf("Expected program to exit with error, but it didn't")
// 		}
// 	}()

// 	main()
// }
