package password

import (
	"os"
	"strings"
	"testing"
)


func TestGetPasswordFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		envVar        string
		envValue      string
		expectedPass  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid env var",
			envVar:       "TEST_PASSWORD",
			envValue:     "mypassword123",
			expectedPass: "mypassword123",
			expectError:  false,
		},
		{
			name:          "empty env var",
			envVar:        "EMPTY_PASSWORD",
			envValue:      "",
			expectError:   true,
			errorContains: "not set or empty",
		},
		{
			name:          "undefined env var",
			envVar:        "UNDEFINED_PASSWORD",
			expectError:   true,
			errorContains: "not set or empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			// Test reading password
			password, err := readPasswordFromEnv(tt.envVar)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if password != tt.expectedPass {
					t.Errorf("Expected password '%s', got '%s'", tt.expectedPass, password)
				}
			}
		})
	}
}

func TestValidatePasswordOptions(t *testing.T) {
	tests := []struct {
		name          string
		opts          Options
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid options with env",
			opts:        Options{PasswordEnv: "TEST_VAR"},
			expectError: false,
		},
		{
			name:        "empty options",
			opts:        Options{},
			expectError: false,
		},
		{
			name:          "empty env var name",
			opts:          Options{PasswordEnv: "   "},
			expectError:   true,
			errorContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordOptions(tt.opts)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetPasswordPriority(t *testing.T) {
	// Set environment variable
	envPassword := "env-password"
	envVar := "TEST_PASSWORD_PRIORITY"
	os.Setenv(envVar, envPassword)
	defer os.Unsetenv(envVar)

	tests := []struct {
		name         string
		opts         Options
		expectedPass string
	}{
		{
			name: "env used when provided",
			opts: Options{
				PasswordEnv: envVar,
			},
			expectedPass: envPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock interactive prompt to avoid blocking
			// Note: This test assumes env is provided, so interactive won't be called
			
			password, err := GetPassword(tt.opts)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if password != tt.expectedPass {
				t.Errorf("Expected password '%s', got '%s'", tt.expectedPass, password)
			}
			
			// Test that password can be cleared
			ClearPassword(&password)
			if password != "" {
				t.Errorf("Password was not cleared, still contains: %s", password)
			}
		})
	}
}

func TestClearPassword(t *testing.T) {
	tests := []struct {
		name     string
		password *string
	}{
		{
			name:     "clear valid password",
			password: func() *string { s := "secret123"; return &s }(),
		},
		{
			name:     "clear empty password",
			password: func() *string { s := ""; return &s }(),
		},
		{
			name:     "clear nil password",
			password: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClearPassword(tt.password)
			
			if tt.password != nil && *tt.password != "" {
				t.Errorf("Password was not cleared, still contains: %s", *tt.password)
			}
		})
	}
}

