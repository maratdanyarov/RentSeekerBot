package config

import (
	"os"
	"testing"
)

// TestGetEnv tests the GetEnv function.
func TestGetEnv(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_ENV_VAR", "test_value")

	// Test GetEnv with existing variable
	value := GetEnv("TEST_ENV_VAR")
	if value != "test_value" {
		t.Errorf("GetEnv() = %v, want %v", value, "test_value")
	}

	// Test GetEnv with non-existent variable
	value = GetEnv("NON_EXISTENT_VAR")
	if value != "" {
		t.Errorf("GetEnv() for non-existent variable = %v, want empty string", value)
	}
}

// TestLoadConfig tests the LoadConfig function.
func TestLoadConfig(t *testing.T) {
	// Create a temporary .env file
	tempEnvContent := []byte("TEST_LOAD_CONFIG=success\n")
	err := os.WriteFile(".env", tempEnvContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create temporary .env file: %v", err)
	}
	defer os.Remove(".env") // Clean up after the test

	// Call LoadConfig
	LoadConfig()

	// Check if the environment variable was loaded
	value := os.Getenv("TEST_LOAD_CONFIG")
	if value != "success" {
		t.Errorf("LoadConfig() failed to load environment variable. Got %v, want %v", value, "success")
	}
}
