// Package config provides functionality for loading and accessing environment variables.
package config

import (
	"os"
	"testing"
)

// TestGetEnv tests the GetEnv function to ensure it correctly retrieves environment variables.
func TestGetEnv(t *testing.T) {
	// Set up test cases
	testCases := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{"ExistingKey", "TEST_KEY", "test_value", "test_value"},
		{"NonExistentKey", "NON_EXISTENT_KEY", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable if a value is provided
			if tc.value != "" {
				os.Setenv(tc.key, tc.value)
				defer os.Unsetenv(tc.key)
			}

			// Call the function being tested
			result := GetEnv(tc.key)

			// Check if the result matches the expected value
			if result != tc.expected {
				t.Errorf("GetEnv(%s) = %s; want %s", tc.key, result, tc.expected)
			}
		})
	}
}
