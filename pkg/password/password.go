package password

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Options contains password input configuration.
//
// Sources are tried in this order: Stdin, PasswordEnv, FallbackEnvs, then the
// interactive prompt. Stdin and PasswordEnv are explicit requests, so an empty
// result there is an error rather than a fall-through; FallbackEnvs are
// conventions, so an unset one is skipped silently.
type Options struct {
	PasswordEnv string // Environment variable name the caller asked for
	// FallbackEnvs are tried in order when PasswordEnv is empty. The CLI
	// passes GOINGENV_PASSWORD_<ENV> then GOINGENV_PASSWORD.
	FallbackEnvs []string
	// Stdin reads one line from standard input and never prompts. It wins
	// over every other source. Only the first line is consumed, so a command
	// that hands stdin to a child process leaves the rest intact.
	Stdin bool
	// Confirm asks for the password a second time at the interactive prompt
	// and rejects a mismatch. Pack sets it: a typo there produces an archive
	// nobody can open. It has no effect when the password comes from
	// PasswordEnv, which is not typed.
	Confirm bool
}

// stdinReader is what Stdin reads from. It is a variable so tests can supply
// scripted input without a pipe.
var stdinReader io.Reader = os.Stdin

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

// ResolveNonInteractive tries every source that does not need a terminal:
// stdin, then PasswordEnv, then FallbackEnvs. ok is false when none of them
// supplied a password and nothing went wrong, so the caller can decide
// whether to prompt.
func ResolveNonInteractive(opts Options) (password string, ok bool, err error) {
	if opts.Stdin {
		password, err = readPasswordFromStdin(stdinReader)
		if err != nil {
			return "", false, fmt.Errorf("failed to read password from stdin: %w", err)
		}
		return password, true, nil
	}

	if opts.PasswordEnv != "" {
		password, err = readPasswordFromEnv(opts.PasswordEnv)
		if err != nil {
			return "", false, fmt.Errorf("failed to read password from environment: %w", err)
		}
		warnEnv(opts.PasswordEnv)
		return password, true, nil
	}

	for _, name := range opts.FallbackEnvs {
		if value := os.Getenv(name); value != "" {
			warnEnv(name)
			return value, true, nil
		}
	}

	return "", false, nil
}

// GetPassword retrieves password using the specified options
// Priority order: Stdin -> PasswordEnv -> FallbackEnvs -> Interactive prompt
func GetPassword(opts Options) (string, error) {
	password, ok, err := ResolveNonInteractive(opts)
	if err != nil {
		return "", err
	}
	if ok {
		return password, nil
	}

	// Fall back to interactive prompt
	return readPasswordInteractively(opts.Confirm)
}

func warnEnv(name string) {
	fmt.Fprintf(os.Stderr, "[!] Using the password in %s; environment variables are visible to other processes\n", name)
}

// readPasswordFromStdin reads up to the first newline. It reads one byte at
// a time on purpose: a buffered reader would swallow input past the newline
// that belongs to whatever the command runs next.
func readPasswordFromStdin(r io.Reader) (string, error) {
	var line []byte
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n == 1 {
			if buf[0] == '\n' {
				break
			}
			line = append(line, buf[0])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	password := strings.TrimSuffix(string(line), "\r")
	if password == "" {
		return "", fmt.Errorf("no password on stdin")
	}
	return password, nil
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
