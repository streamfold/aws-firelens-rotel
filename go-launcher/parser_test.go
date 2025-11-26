package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFluentBitConfig(t *testing.T) {
	// Create a temporary config file for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.conf")

	configContent := `[INPUT]
    Name tcp
    Listen 127.0.0.1
    Port 8877
    Tag firelens-healthcheck
[INPUT]
    Name forward
    Mem_Buf_Limit 25MB
    unix_path /var/run/fluent.sock
[INPUT]
    Name forward
    Listen 127.0.0.1
    Port 24224
[FILTER]
    Name record_modifier
    Match *
    Record ecs_cluster firelenstest
    Record ecs_task_arn arn:aws:ecs:us-east-2:279234357137:task/firelenstest/72aa3d1989dc4561b975631f36170c09
    Record ecs_task_definition firelens-test1:8
[OUTPUT]
    Name null
    Match firelens-healthcheck
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Parse the config
	config, err := ParseFluentBitConfig(configPath)
	if err != nil {
		t.Fatalf("ParseFluentBitConfig failed: %v", err)
	}

	// Verify ReceiverEndpoint
	expectedEndpoint := "127.0.0.1:24224"
	if config.ReceiverEndpoint != expectedEndpoint {
		t.Errorf("ReceiverEndpoint = %q, want %q", config.ReceiverEndpoint, expectedEndpoint)
	}

	// Verify ReceiverSocket
	expectedSocket := "/var/run/fluent.sock"
	if config.ReceiverSocket != expectedSocket {
		t.Errorf("ReceiverSocket = %q, want %q", config.ReceiverSocket, expectedSocket)
	}

	// Verify ResourceAttributes
	expectedAttrs := "ecs_cluster=firelenstest,ecs_task_arn=arn:aws:ecs:us-east-2:279234357137:task/firelenstest/72aa3d1989dc4561b975631f36170c09,ecs_task_definition=firelens-test1:8"
	if config.ResourceAttributes != expectedAttrs {
		t.Errorf("ResourceAttributes = %q, want %q", config.ResourceAttributes, expectedAttrs)
	}
}

func TestParseFluentBitConfig_SocketOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.conf")

	configContent := `[INPUT]
    Name forward
    unix_path /var/run/custom.sock
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := ParseFluentBitConfig(configPath)
	if err != nil {
		t.Fatalf("ParseFluentBitConfig failed: %v", err)
	}

	if config.ReceiverSocket != "/var/run/custom.sock" {
		t.Errorf("ReceiverSocket = %q, want %q", config.ReceiverSocket, "/var/run/custom.sock")
	}

	if config.ReceiverEndpoint != "" {
		t.Errorf("ReceiverEndpoint = %q, want empty string", config.ReceiverEndpoint)
	}
}

func TestParseFluentBitConfig_EndpointOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.conf")

	configContent := `[INPUT]
    Name forward
    Listen 0.0.0.0
    Port 9999
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := ParseFluentBitConfig(configPath)
	if err != nil {
		t.Fatalf("ParseFluentBitConfig failed: %v", err)
	}

	if config.ReceiverEndpoint != "0.0.0.0:9999" {
		t.Errorf("ReceiverEndpoint = %q, want %q", config.ReceiverEndpoint, "0.0.0.0:9999")
	}

	if config.ReceiverSocket != "" {
		t.Errorf("ReceiverSocket = %q, want empty string", config.ReceiverSocket)
	}
}

func TestParseFluentBitConfig_NoRecordModifier(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.conf")

	configContent := `[INPUT]
    Name forward
    Listen 127.0.0.1
    Port 24224
[OUTPUT]
    Name null
    Match *
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := ParseFluentBitConfig(configPath)
	if err != nil {
		t.Fatalf("ParseFluentBitConfig failed: %v", err)
	}

	if config.ResourceAttributes != "" {
		t.Errorf("ResourceAttributes = %q, want empty string", config.ResourceAttributes)
	}
}

func TestParseFluentBitConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.conf")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := ParseFluentBitConfig(configPath)
	if err != nil {
		t.Fatalf("ParseFluentBitConfig failed: %v", err)
	}

	if config.ReceiverEndpoint != "" || config.ReceiverSocket != "" || config.ResourceAttributes != "" {
		t.Errorf("Expected all fields to be empty for empty config file")
	}
}

func TestParseFluentBitConfig_NonexistentFile(t *testing.T) {
	_, err := ParseFluentBitConfig("/nonexistent/path/to/config.conf")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestParseFluentBitConfig_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.conf")

	configContent := `# This is a comment
[INPUT]
    # Another comment
    Name forward
    Listen 127.0.0.1
    Port 24224
    # Inline comment should be handled
[FILTER]
    Name record_modifier
    Match *
    # Comment before record
    Record service myservice
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := ParseFluentBitConfig(configPath)
	if err != nil {
		t.Fatalf("ParseFluentBitConfig failed: %v", err)
	}

	if config.ReceiverEndpoint != "127.0.0.1:24224" {
		t.Errorf("ReceiverEndpoint = %q, want %q", config.ReceiverEndpoint, "127.0.0.1:24224")
	}

	if config.ResourceAttributes != "service=myservice" {
		t.Errorf("ResourceAttributes = %q, want %q", config.ResourceAttributes, "service=myservice")
	}
}
