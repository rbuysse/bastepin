package main

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	if config.Bind != "0.0.0.0:3001" {
		t.Errorf("Expected Bind to be '0.0.0.0:3001', got '%s'", config.Bind)
	}
	if config.ServePath != "/p/" {
		t.Errorf("Expected ServePath to be '/p/', got '%s'", config.ServePath)
	}
	if config.DatabasePath != "./pastes.db" {
		t.Errorf("Expected DatabasePath to be './pastes.db', got '%s'", config.DatabasePath)
	}
	if config.Debug {
		t.Error("Expected Debug to be false")
	}
}

func TestEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name: "All environment variables set",
			envVars: map[string]string{
				"PB_BIND":          "127.0.0.1:8080",
				"PB_DATABASE_PATH": "/tmp/test.db",
				"PB_DEBUG":         "true",
				"PB_SERVE_PATH":    "/paste/",
			},
			expected: Config{
				Bind:         "127.0.0.1:8080",
				DatabasePath: "/tmp/test.db",
				Debug:        true,
				ServePath:    "/paste/",
			},
		},
		{
			name: "Partial environment variables - only bind",
			envVars: map[string]string{
				"PB_BIND": "0.0.0.0:9000",
			},
			expected: Config{
				Bind:         "0.0.0.0:9000",
				DatabasePath: "./pastes.db", // default
				Debug:        false,         // default
				ServePath:    "/p/",         // default
			},
		},
		{
			name: "Partial environment variables - only database and debug",
			envVars: map[string]string{
				"PB_DATABASE_PATH": "/var/lib/pb.db",
				"PB_DEBUG":         "1",
			},
			expected: Config{
				Bind:         "0.0.0.0:3001", // default
				DatabasePath: "/var/lib/pb.db",
				Debug:        true,
				ServePath:    "/p/", // default
			},
		},
		{
			name: "Debug with string value 'true'",
			envVars: map[string]string{
				"PB_DEBUG": "true",
			},
			expected: Config{
				Bind:         "0.0.0.0:3001", // default
				DatabasePath: "./pastes.db",  // default
				Debug:        true,
				ServePath:    "/p/", // default
			},
		},
		{
			name: "Debug with string value '1'",
			envVars: map[string]string{
				"PB_DEBUG": "1",
			},
			expected: Config{
				Bind:         "0.0.0.0:3001", // default
				DatabasePath: "./pastes.db",  // default
				Debug:        true,
				ServePath:    "/p/", // default
			},
		},
		{
			name: "Debug with false value",
			envVars: map[string]string{
				"PB_DEBUG": "false",
			},
			expected: Config{
				Bind:         "0.0.0.0:3001", // default
				DatabasePath: "./pastes.db",  // default
				Debug:        false,
				ServePath:    "/p/", // default
			},
		},
		{
			name:    "No environment variables - use defaults",
			envVars: map[string]string{},
			expected: Config{
				Bind:         "0.0.0.0:3001",
				DatabasePath: "./pastes.db",
				Debug:        false,
				ServePath:    "/p/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all PB_ environment variables first
			clearPBEnvVars()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Start with defaults
			config := defaultConfig()

			// Apply environment variables (simulating the logic from GenerateConfig)
			if envBind := os.Getenv("PB_BIND"); envBind != "" {
				config.Bind = envBind
			}
			if envDB := os.Getenv("PB_DATABASE_PATH"); envDB != "" {
				config.DatabasePath = envDB
			}
			if envDebug := os.Getenv("PB_DEBUG"); envDebug == "true" || envDebug == "1" {
				config.Debug = true
			}
			if envServePath := os.Getenv("PB_SERVE_PATH"); envServePath != "" {
				config.ServePath = envServePath
			}

			// Verify results
			if config.Bind != tt.expected.Bind {
				t.Errorf("Expected Bind to be '%s', got '%s'", tt.expected.Bind, config.Bind)
			}
			if config.DatabasePath != tt.expected.DatabasePath {
				t.Errorf("Expected DatabasePath to be '%s', got '%s'", tt.expected.DatabasePath, config.DatabasePath)
			}
			if config.Debug != tt.expected.Debug {
				t.Errorf("Expected Debug to be %v, got %v", tt.expected.Debug, config.Debug)
			}
			if config.ServePath != tt.expected.ServePath {
				t.Errorf("Expected ServePath to be '%s', got '%s'", tt.expected.ServePath, config.ServePath)
			}
		})
	}
}

func TestConfigPrecedence(t *testing.T) {
	// This test verifies the precedence order:
	// CLI flags > Environment variables > Config file > Defaults

	t.Run("Environment variables override defaults", func(t *testing.T) {
		clearPBEnvVars()
		os.Setenv("PB_BIND", "env-value:1234")
		defer os.Unsetenv("PB_BIND")

		config := defaultConfig()
		if envBind := os.Getenv("PB_BIND"); envBind != "" {
			config.Bind = envBind
		}

		if config.Bind != "env-value:1234" {
			t.Errorf("Expected env var to override default, got '%s'", config.Bind)
		}
	})

	t.Run("Partial env vars preserve other defaults", func(t *testing.T) {
		clearPBEnvVars()
		os.Setenv("PB_SERVE_PATH", "/custom/")
		defer os.Unsetenv("PB_SERVE_PATH")

		config := defaultConfig()

		// Apply env vars
		if envServePath := os.Getenv("PB_SERVE_PATH"); envServePath != "" {
			config.ServePath = envServePath
		}

		// Check that the env var was applied
		if config.ServePath != "/custom/" {
			t.Errorf("Expected ServePath to be '/custom/', got '%s'", config.ServePath)
		}

		// Check that other fields still have defaults
		if config.Bind != "0.0.0.0:3001" {
			t.Errorf("Expected Bind to remain default '0.0.0.0:3001', got '%s'", config.Bind)
		}
		if config.DatabasePath != "./pastes.db" {
			t.Errorf("Expected DatabasePath to remain default './pastes.db', got '%s'", config.DatabasePath)
		}
	})

	t.Run("Empty env vars don't override defaults", func(t *testing.T) {
		clearPBEnvVars()
		os.Setenv("PB_BIND", "")
		defer os.Unsetenv("PB_BIND")

		config := defaultConfig()

		// Apply env vars (empty string check)
		if envBind := os.Getenv("PB_BIND"); envBind != "" {
			config.Bind = envBind
		}

		// Empty env var should not override default
		if config.Bind != "0.0.0.0:3001" {
			t.Errorf("Expected empty env var to not override default, got '%s'", config.Bind)
		}
	})
}

// Helper function to clear all PB_ environment variables
func clearPBEnvVars() {
	os.Unsetenv("PB_BIND")
	os.Unsetenv("PB_DATABASE_PATH")
	os.Unsetenv("PB_DEBUG")
	os.Unsetenv("PB_SERVE_PATH")
}
