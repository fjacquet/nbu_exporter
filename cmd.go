package main

type context struct {
	Debug bool
}

// ConfigCommand type to add to command line parser
// ConfigCommand is a command-line struct that holds the configuration file path.
// The Path field specifies the file path for the configuration file.
type ConfigCommand struct {
	// Path file path for configuration file
	Path string `arg optional name:"path" help:"Paths to list." type:"path"`
}

// Run in the case of a configuration parameter
func (l *ConfigCommand) Run(ctx *context) error {
	// fmt.Println("config file is ", l.Path)
	ConfigFile = l.Path
	return nil
}
