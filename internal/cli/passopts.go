package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"goingenv/pkg/password"
)

// DefaultPasswordEnv is the environment variable every command consults
// when --password-env is not given.
const DefaultPasswordEnv = "GOINGENV_PASSWORD"

// PassOpts holds the password-source flags shared by every command that
// opens an archive.
type PassOpts struct {
	PassEnv   string // --password-env
	PassStdin bool   // --password-stdin
}

// addPasswordFlags registers --password-env and --password-stdin.
func addPasswordFlags(cmd *cobra.Command) {
	cmd.Flags().String("password-env", "", "Read the password from this environment variable (default: "+DefaultPasswordEnv+", then "+DefaultPasswordEnv+"_<ENV>)")
	cmd.Flags().Bool("password-stdin", false, "Read the password from the first line of standard input; never prompts")
}

// parsePassOpts reads the password-source flags.
func parsePassOpts(cmd *cobra.Command) (PassOpts, error) {
	var p PassOpts
	var err error
	if p.PassEnv, err = cmd.Flags().GetString("password-env"); err != nil {
		return p, fmt.Errorf("failed to get password-env flag: %w", err)
	}
	if p.PassStdin, err = cmd.Flags().GetBool("password-stdin"); err != nil {
		return p, fmt.Errorf("failed to get password-stdin flag: %w", err)
	}
	return p, nil
}

// envPasswordVar is the per-environment variable name: GOINGENV_PASSWORD_PROD
// for "prod". Hyphens become underscores because a shell cannot export a
// name containing one, so "prod-eu" and "prod_eu" share a variable.
func envPasswordVar(envName string) string {
	return DefaultPasswordEnv + "_" + strings.ToUpper(strings.ReplaceAll(envName, "-", "_"))
}

// passwordOptionsFor builds the password.Options for a command. With an
// environment name the per-environment variable is tried before the default.
func passwordOptionsFor(p PassOpts, envName string, confirm bool) password.Options {
	fallbacks := []string{DefaultPasswordEnv}
	if envName != "" {
		fallbacks = []string{envPasswordVar(envName), DefaultPasswordEnv}
	}
	return password.Options{
		PasswordEnv:  p.PassEnv,
		FallbackEnvs: fallbacks,
		Stdin:        p.PassStdin,
		Confirm:      confirm && !p.PassStdin,
	}
}

// getPass retrieves the password with a cleanup function. confirm asks for
// the password a second time at an interactive prompt; it has no effect when
// the password comes from a variable or stdin, which are not typed.
func getPass(p PassOpts, envName string, confirm bool) (key string, cleanup func(), err error) {
	opts := passwordOptionsFor(p, envName, confirm)
	if validateErr := password.ValidatePasswordOptions(opts); validateErr != nil {
		return "", nil, fmt.Errorf("invalid password options: %w", validateErr)
	}

	key, err = password.GetPassword(opts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get password: %w", err)
	}

	cleanup = func() { password.ClearPassword(&key) }
	return key, cleanup, nil
}
