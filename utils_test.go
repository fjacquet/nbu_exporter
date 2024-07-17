package main

// type mockConfig struct {
// 	Field1 string
// 	Field2 int
// }

// func TestReadFile(t *testing.T) {
// 	tmpFile, err := os.CreateTemp("", "test.yml")
// 	if err != nil {
// 		t.Fatalf("failed to create temp file: %v", err)
// 	}
// 	defer os.Remove(tmpFile.Name())

// 	config := mockConfig{
// 		Field1: "value1",
// 		Field2: 42,
// 	}

// 	data, err := yaml.Marshal(config)
// 	if err != nil {
// 		t.Fatalf("failed to marshal config: %v", err)
// 	}

// 	if _, err := tmpFile.Write(data); err != nil {
// 		t.Fatalf("failed to write to temp file: %v", err)
// 	}

// 	var cfg mockConfig
// 	ReadFile(s&cfg, tmpFile.Name())

// 	if cfg.Field1 != config.Field1 {
// 		t.Errorf("unexpected Field1 value: got %s, want %s", cfg.Field1, config.Field1)
// 	}

// 	if cfg.Field2 != config.Field2 {
// 		t.Errorf("unexpected Field2 value: got %d, want %d", cfg.Field2, config.Field2)
// 	}
// }

// func TestReadFileNonExistentFile(t *testing.T) {
// 	var cfg mockConfig
// 	err := ReadFile(&cfg, "non-existent.yml")
// 	if err == nil {
// 		t.Error("expected error for non-existent file, but got nil")
// 	}
// }

// func TestReadFileInvalidYAML(t *testing.T) {
// 	tmpFile, err := os.CreateTemp("", "test.yml")
// 	if err != nil {
// 		t.Fatalf("failed to create temp file: %v", err)
// 	}
// 	defer os.Remove(tmpFile.Name())

// 	if _, err := tmpFile.WriteString("invalid yaml"); err != nil {
// 		t.Fatalf("failed to write to temp file: %v", err)
// 	}

// 	var cfg mockConfig
// 	err = ReadFile(&cfg, tmpFile.Name())
// 	if err == nil {
// 		t.Error("expected error for invalid YAML, but got nil")
// 	}
// }
