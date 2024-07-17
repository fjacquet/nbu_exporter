package main

// func TestConfigUnmarshalYAML(t *testing.T) {
// 	yamlData := []byte(`
// server:
//   port: "8080"
//   host: "localhost"
//   uri: "/metrics"
//   scrappingInterval: "30s"
//   logName: "app.log"

// nbuserver:
//   port: "9090"
//   scheme: "https"
//   uri: "/api/v1"
//   domain: "example.com"
//   domainType: "vCenter"
//   host: "nbu.example.com"
//   apiKey: "secret-key"
//   contentType: "application/json"
// `)

// 	var cfg Config
// 	err := yaml.UnmarshalStrict(yamlData, &cfg)
// 	if err != nil {
// 		t.Errorf("Failed to unmarshal YAML data: %v", err)
// 	}

// 	if cfg.Server.Port != "8080" {
// 		t.Errorf("Incorrect Server.Port value: %s", cfg.Server.Port)
// 	}

// 	if cfg.Server.Host != "localhost" {
// 		t.Errorf("Incorrect Server.Host value: %s", cfg.Server.Host)
// 	}

// 	if cfg.Server.URI != "/metrics" {
// 		t.Errorf("Incorrect Server.URI value: %s", cfg.Server.URI)
// 	}

// 	if cfg.Server.ScrappingInterval != "30s" {
// 		t.Errorf("Incorrect Server.ScrappingInterval value: %s", cfg.Server.ScrappingInterval)
// 	}

// 	if cfg.Server.LogName != "app.log" {
// 		t.Errorf("Incorrect Server.LogName value: %s", cfg.Server.LogName)
// 	}

// 	if cfg.NbuServer.Port != "9090" {
// 		t.Errorf("Incorrect NbuServer.Port value: %s", cfg.NbuServer.Port)
// 	}

// 	if cfg.NbuServer.Scheme != "https" {
// 		t.Errorf("Incorrect NbuServer.Scheme value: %s", cfg.NbuServer.Scheme)
// 	}

// 	if cfg.NbuServer.URI != "/api/v1" {
// 		t.Errorf("Incorrect NbuServer.URI value: %s", cfg.NbuServer.URI)
// 	}

// 	if cfg.NbuServer.Domain != "example.com" {
// 		t.Errorf("Incorrect NbuServer.Domain value: %s", cfg.NbuServer.Domain)
// 	}

// 	if cfg.NbuServer.DomainType != "vCenter" {
// 		t.Errorf("Incorrect NbuServer.DomainType value: %s", cfg.NbuServer.DomainType)
// 	}

// 	if cfg.NbuServer.Host != "nbu.example.com" {
// 		t.Errorf("Incorrect NbuServer.Host value: %s", cfg.NbuServer.Host)
// 	}

// 	if cfg.NbuServer.APIKey != "secret-key" {
// 		t.Errorf("Incorrect NbuServer.APIKey value: %s", cfg.NbuServer.APIKey)
// 	}

// 	if cfg.NbuServer.ContentType != "application/json" {
// 		t.Errorf("Incorrect NbuServer.ContentType value: %s", cfg.NbuServer.ContentType)
// 	}
// }

// func TestConfigUnmarshalYAMLWithMissingFields(t *testing.T) {
// 	yamlData := []byte(`
// server:
//   port: "8080"
//   host: "localhost"

// nbuserver:
//   port: "9090"
//   scheme: "https"
// `)

// 	var cfg Config
// 	err := yaml.UnmarshalStrict(yamlData, &cfg)
// 	if err != nil {
// 		t.Errorf("Failed to unmarshal YAML data: %v", err)
// 	}

// 	if cfg.Server.Port != "8080" {
// 		t.Errorf("Incorrect Server.Port value: %s", cfg.Server.Port)
// 	}

// 	if cfg.Server.Host != "localhost" {
// 		t.Errorf("Incorrect Server.Host value: %s", cfg.Server.Host)
// 	}

// 	if cfg.Server.URI != "" {
// 		t.Errorf("Unexpected Server.URI value: %s", cfg.Server.URI)
// 	}

// 	if cfg.Server.ScrappingInterval != "" {
// 		t.Errorf("Unexpected Server.ScrappingInterval value: %s", cfg.Server.ScrappingInterval)
// 	}

// 	if cfg.Server.LogName != "" {
// 		t.Errorf("Unexpected Server.LogName value: %s", cfg.Server.LogName)
// 	}

// 	if cfg.NbuServer.Port != "9090" {
// 		t.Errorf("Incorrect NbuServer.Port value: %s", cfg.NbuServer.Port)
// 	}

// 	if cfg.NbuServer.Scheme != "https" {
// 		t.Errorf("Incorrect NbuServer.Scheme value: %s", cfg.NbuServer.Scheme)
// 	}

// 	if cfg.NbuServer.URI != "" {
// 		t.Errorf("Unexpected NbuServer.URI value: %s", cfg.NbuServer.URI)
// 	}

// 	if cfg.NbuServer.Domain != "" {
// 		t.Errorf("Unexpected NbuServer.Domain value: %s", cfg.NbuServer.Domain)
// 	}

// 	if cfg.NbuServer.DomainType != "" {
// 		t.Errorf("Unexpected NbuServer.DomainType value: %s", cfg.NbuServer.DomainType)
// 	}

// 	if cfg.NbuServer.Host != "" {
// 		t.Errorf("Unexpected NbuServer.Host value: %s", cfg.NbuServer.Host)
// 	}

// 	if cfg.NbuServer.APIKey != "" {
// 		t.Errorf("Unexpected NbuServer.APIKey value: %s", cfg.NbuServer.APIKey)
// 	}

// 	if cfg.NbuServer.ContentType != "" {
// 		t.Errorf("Unexpected NbuServer.ContentType value: %s", cfg.NbuServer.ContentType)
// 	}
// }

// func TestConfigUnmarshalYAMLWithInvalidData(t *testing.T) {
// 	yamlData := []byte(`
// server:
//   port: 8080
//   host: localhost

// nbuserver:
//   port: 9090
//   scheme: https
// `)

// 	var cfg Config
// 	err := yaml.UnmarshalStrict(yamlData, &cfg)
// 	if err == nil {
// 		t.Error("Expected error when unmarshaling invalid YAML data, but got nil")
// 	}
// }
