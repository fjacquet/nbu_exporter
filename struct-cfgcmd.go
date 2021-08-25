package main

// CfgCmd type to add to command line parser
type ConfigCommand struct {
	// Path file path for configuration file
	Path string `arg optional name:"path" help:"Paths to list." type:"path"`
}
