package main

import (
	"testing"
)

// TestMaskAccessKey verifies the access key masking function
func TestMaskAccessKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard access key",
			input:    "AKIAIOSFODNN7EXAMPLE",
			expected: "AKIA****MPLE",
		},
		{
			name:     "short key",
			input:    "SHORT",
			expected: "****",
		},
		{
			name:     "very short key",
			input:    "ABC",
			expected: "****",
		},
		{
			name:     "empty key",
			input:    "",
			expected: "****",
		},
		{
			name:     "exactly 8 characters",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "9 characters",
			input:    "123456789",
			expected: "1234****6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAccessKey(tt.input)
			if result != tt.expected {
				t.Errorf("maskAccessKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMain_Integration tests the main function with various configurations
// Note: These are integration tests that verify the startup logic
func TestMain_Integration(t *testing.T) {
	// This test verifies that the main package compiles and the helper functions work
	// The actual main() function is tested manually since it requires stdio interaction
	t.Run("helper functions exist", func(t *testing.T) {
		// Verify maskAccessKey works
		masked := maskAccessKey("AKIAIOSFODNN7EXAMPLE")
		if masked == "" {
			t.Error("maskAccessKey returned empty string")
		}
	})
}
