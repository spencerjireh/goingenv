package password

import (
	"io"
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

// scriptSecrets replaces the terminal read with a fixed sequence of answers
// and restores it when the test ends.
func scriptSecrets(t *testing.T, answers ...string) {
	t.Helper()

	saved := readSecret
	t.Cleanup(func() { readSecret = saved })

	readSecret = func() (string, error) {
		if len(answers) == 0 {
			t.Fatal("the prompt asked for more input than the test scripted")
		}
		next := answers[0]
		answers = answers[1:]
		return next, nil
	}
}

func TestReadPasswordInteractively(t *testing.T) {
	tests := []struct {
		name          string
		confirm       bool
		answers       []string
		want          string
		errorContains string
	}{
		{name: "single entry", answers: []string{"hunter2"}, want: "hunter2"},
		{name: "single entry ignores confirm answer", answers: []string{"hunter2", "other"}, want: "hunter2"},
		{name: "confirm matches", confirm: true, answers: []string{"hunter2", "hunter2"}, want: "hunter2"},
		{name: "confirm mismatch", confirm: true, answers: []string{"hunter2", "hunter3"}, errorContains: "do not match"},
		{name: "empty first entry", confirm: true, answers: []string{""}, errorContains: "cannot be empty"},
		{name: "empty confirmation", confirm: true, answers: []string{"hunter2", ""}, errorContains: "do not match"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptSecrets(t, tt.answers...)

			got, err := readPasswordInteractively(tt.confirm)

			if tt.errorContains != "" {
				if err == nil {
					t.Fatalf("got password %q, want an error containing %q", got, tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err, tt.errorContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestConfirmDoesNotApplyToEnv pins that Confirm only affects the prompt: an
// env-var password is not typed, so there is nothing to confirm.
func TestConfirmDoesNotApplyToEnv(t *testing.T) {
	t.Setenv("TEST_CONFIRM_ENV", "from-env")
	scriptSecrets(t) // any prompt would fail the test

	got, err := GetPassword(Options{PasswordEnv: "TEST_CONFIRM_ENV", Confirm: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from-env" {
		t.Errorf("got %q, want the env value", got)
	}
}

// scriptStdin replaces standard input with fixed content and returns the
// reader so a test can check what was left unread.
func scriptStdin(t *testing.T, content string) *strings.Reader {
	t.Helper()
	saved := stdinReader
	t.Cleanup(func() { stdinReader = saved })
	r := strings.NewReader(content)
	stdinReader = r
	return r
}

func TestStdinWinsOverEnvAndNeverPrompts(t *testing.T) {
	t.Setenv("TEST_STDIN_ENV", "from-env")
	scriptSecrets(t) // any prompt would fail the test
	scriptStdin(t, "from-stdin\n")

	got, err := GetPassword(Options{Stdin: true, PasswordEnv: "TEST_STDIN_ENV", Confirm: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from-stdin" {
		t.Errorf("got %q, want the stdin value", got)
	}
}

func TestStdinReadsOneLineOnly(t *testing.T) {
	r := scriptStdin(t, "secret\r\nrest of input\n")

	got, err := GetPassword(Options{Stdin: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "secret" {
		t.Errorf("got %q, want secret without the CR", got)
	}
	rest, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(rest) != "rest of input\n" {
		t.Errorf("remaining stdin = %q, want it untouched", rest)
	}
}

func TestStdinWithoutNewline(t *testing.T) {
	scriptStdin(t, "secret")
	got, err := GetPassword(Options{Stdin: true})
	if err != nil || got != "secret" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestStdinEmptyIsAnError(t *testing.T) {
	scriptStdin(t, "\n")
	scriptSecrets(t)
	if _, err := GetPassword(Options{Stdin: true}); err == nil {
		t.Fatal("expected an error for an empty stdin line")
	}
}

func TestFallbackEnvsOrder(t *testing.T) {
	scriptSecrets(t)
	t.Setenv("TEST_FB_PROD", "prod-pw")
	t.Setenv("TEST_FB_DEFAULT", "default-pw")

	got, err := GetPassword(Options{FallbackEnvs: []string{"TEST_FB_UNSET", "TEST_FB_PROD", "TEST_FB_DEFAULT"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "prod-pw" {
		t.Errorf("got %q, want the first set fallback", got)
	}
}

func TestExplicitEnvUnsetIsAnErrorEvenWithFallbacks(t *testing.T) {
	t.Setenv("TEST_FB_DEFAULT2", "default-pw")
	_, err := GetPassword(Options{PasswordEnv: "TEST_EXPLICIT_UNSET", FallbackEnvs: []string{"TEST_FB_DEFAULT2"}})
	if err == nil {
		t.Fatal("an explicitly named but unset variable must be an error, not a fall-through")
	}
}

func TestResolveNonInteractiveDoesNotPrompt(t *testing.T) {
	scriptSecrets(t)
	pw, ok, err := ResolveNonInteractive(Options{FallbackEnvs: []string{"TEST_RNI_UNSET"}})
	if err != nil || ok || pw != "" {
		t.Errorf("got (%q, %v, %v), want (\"\", false, nil)", pw, ok, err)
	}
}
