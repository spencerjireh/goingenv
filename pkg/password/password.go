package password

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Options contains password input configuration
type Options struct {
	PasswordEnv string // Environment variable name
	// Confirm asks for the password a second time at the interactive prompt
	// and rejects a mismatch. Pack sets it: a typo there produces an archive
	// nobody can open. It has no effect when the password comes from
	// PasswordEnv, which is not typed.
	Confirm bool
}

// readSecret reads one hidden line from the terminal. It is a variable so
// tests can substitute scripted answers; the interactive path is otherwise
// untestable without a pty.
var readSecret = func() (string, error) {
	passwordBytes, err := term.ReadPassword(syscall.Stdin)
	fmt.Println() // Add newline after hidden input
	if err != nil {
		return "", err
	}
	return string(passwordBytes), nil
}

// GetPassword retrieves password using the specified options
// Priority order: PasswordEnv -> Interactive prompt
func GetPassword(opts Options) (string, error) {
	var password string
	var err error

	// Try environment variable first
	if opts.PasswordEnv != "" {
		password, err = readPasswordFromEnv(opts.PasswordEnv)
		if err != nil {
			return "", fmt.Errorf("failed to read password from environment: %w", err)
		}
		if password != "" {
			fmt.Fprintf(os.Stderr, "[!] Using the password in %s; environment variables are visible to other processes\n", opts.PasswordEnv)
			return password, nil
		}
	}

	// Fall back to interactive prompt
	return readPasswordInteractively(opts.Confirm)
}

// readPasswordFromEnv reads password from environment variable
func readPasswordFromEnv(envVar string) (string, error) {
	password := os.Getenv(envVar)
	if password == "" {
		return "", fmt.Errorf("environment variable '%s' is not set or empty", envVar)
	}
	return password, nil
}

// readPasswordInteractively prompts user for password with hidden input.
// With confirm set it prompts a second time and requires the entries to match.
func readPasswordInteractively(confirm bool) (string, error) {
	fmt.Print("Enter encryption password: ")
	password, err := readSecret()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	if !confirm {
		return password, nil
	}

	fmt.Print("Confirm encryption password: ")
	again, err := readSecret()
	if err != nil {
		return "", fmt.Errorf("failed to read password confirmation: %w", err)
	}
	if again != password {
		return "", fmt.Errorf("passwords do not match")
	}

	return password, nil
}

// ClearPassword securely clears password from memory
func ClearPassword(password *string) {
	if password == nil {
		return
	}

	// Convert to byte slice and clear each byte
	bytes := []byte(*password)
	for i := range bytes {
		bytes[i] = 0
	}

	// Set string to empty
	*password = ""
}

// ValidatePasswordOptions validates the password options
func ValidatePasswordOptions(opts Options) error {
	// Check that environment variable is valid if specified
	if opts.PasswordEnv != "" {
		if strings.TrimSpace(opts.PasswordEnv) == "" {
			return fmt.Errorf("environment variable name cannot be empty")
		}
	}

	return nil
}
