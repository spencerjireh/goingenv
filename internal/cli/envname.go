package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
	"goingenv/pkg/types"
)

// addEnvFlag registers --env.
func addEnvFlag(cmd *cobra.Command) {
	cmd.Flags().String("env", "", "Environment name: archives are named <env>-<timestamp>.enc and the password is read from "+DefaultPasswordEnv+"_<ENV> first")
}

// parseEnvFlag reads and validates --env.
func parseEnvFlag(cmd *cobra.Command) (string, error) {
	env, err := cmd.Flags().GetString("env")
	if err != nil {
		return "", fmt.Errorf("failed to get env flag: %w", err)
	}
	if err := config.ValidateEnvName(env); err != nil {
		return "", err
	}
	return env, nil
}

// archivesForEnv keeps the archives that belong to env, newest first.
//
// It matches the full default name shape rather than a bare prefix, so env
// "prod" does not claim "prod-eu-<ts>.enc" and a custom -o name never takes
// part in latest-selection. Default names end in a timestamp, so sorting by
// name descending is sorting by age.
func archivesForEnv(archives []string, env string) []string {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(config.ArchivePrefix(env)) + `\d{8}-\d{6}\.enc$`)
	var matches []string
	for _, p := range archives {
		if re.MatchString(filepath.Base(p)) {
			matches = append(matches, p)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	return matches
}

// pickArchive returns the archive a command should open: the one the user
// named, or the newest one for env.
func pickArchive(app *types.App, specified, env string) (string, error) {
	if specified != "" {
		return resolveArchiveArg(specified), nil
	}

	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return "", fmt.Errorf("failed to find archives: %w", err)
	}
	matches := archivesForEnv(archives, env)
	if len(matches) == 0 {
		if env != "" {
			return "", fmt.Errorf("no archives found for environment %q in the %s directory: pack one with --env %s, or use -f to name a file", env, config.GetGoingEnvDir(), env)
		}
		return "", fmt.Errorf("no archives found in the %s directory: use the -f flag to specify one", config.GetGoingEnvDir())
	}
	return matches[0], nil
}

// resolveArchiveArg lets a user name an archive by its bare file name: when
// arg does not exist as given but does under the goingenv directory, the
// latter is returned.
func resolveArchiveArg(arg string) string {
	if _, err := os.Stat(arg); err == nil || filepath.IsAbs(arg) {
		return arg
	}
	candidate := filepath.Join(config.GetGoingEnvDir(), arg)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return arg
}

// describeDecryptError is what a command returns when an archive would not
// open. Only the authentication failure collapses to the generic message,
// since a wrong password and a corrupted file are indistinguishable by
// design. The legacy-format sentinel carries its own remedy and passes
// through bare; other header errors (a newer format version, an unsupported
// KDF or key mode, out-of-range parameters) keep their text so the user is
// told to upgrade rather than to retype the password. Anything else -- a
// read error, a broken tar -- is already descriptive and is returned as is.
func describeDecryptError(err error) error {
	switch {
	case errors.Is(err, types.ErrLegacyArchive):
		return types.ErrLegacyArchive
	case errors.Is(err, types.ErrDecryptFailed):
		return fmt.Errorf("failed to decrypt archive: %w", types.ErrDecryptFailed)
	}
	var cryptoErr *types.CryptoError
	if errors.As(err, &cryptoErr) {
		return fmt.Errorf("cannot open archive: %w", cryptoErr.Err)
	}
	return err
}
