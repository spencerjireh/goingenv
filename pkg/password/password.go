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
			fmt.Fprintf(os.Stderr, "⚠️  Security Warning: Using password from environment variable '%s'\n", opts.PasswordEnv)
			fmt.Fprintf(os.Stderr, "   Environment variables may be visible to other processes\n")
			return password, nil
		}
	}

	// Fall back to interactive prompt
	return readPasswordInteractively()
}

// readPasswordFromEnv reads password from environment variable
func readPasswordFromEnv(envVar string) (string, error) {
	password := os.Getenv(envVar)
	if password == "" {
		return "", fmt.Errorf("environment variable '%s' is not set or empty", envVar)
	}
	return password, nil
}

// readPasswordInteractively prompts user for password with hidden input
func readPasswordInteractively() (string, error) {
	fmt.Print("Enter encryption password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after hidden input

	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	password := string(passwordBytes)
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
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
