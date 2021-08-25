package main

// Run in the case of a configuration parameter
func (l *ConfigCommand) Run(ctx *context) error {
	// fmt.Println("config file is ", l.Path)
	ConfigFile = l.Path
	return nil
}
