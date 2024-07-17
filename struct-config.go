package main

// Config represents the configuration for the application.
// It includes settings for the server and the NBU server.
type Config struct {
	Server struct {
		Port              string `yaml:"port"`
		Host              string `yaml:"host"`
		URI               string `yaml:"uri"`
		ScrappingInterval string `yaml:"scrappingInterval"`
		LogName           string `yaml:"logName"`
	} `yaml:"server"`

	NbuServer struct {
		Port        string `yaml:"port"`
		Scheme      string `yaml:"scheme"`
		URI         string `yaml:"uri"`
		Domain      string `yaml:"domain"`
		DomainType  string `yaml:"domainType"`
		Host        string `yaml:"host"`
		APIKey      string `yaml:"apiKey"`
		ContentType string `yaml:"contentType"`
	} `yaml:"nbuserver"`
}
