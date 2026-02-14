package cli

import (
	"fmt"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"goingenv/internal/config"
	"goingenv/pkg/password"
	"goingenv/pkg/types"
)

// UnpackOpts holds parsed unpack command flags
type UnpackOpts struct {
	Archive   string
	Target    string
	PassEnv   string
	Overwrite bool
	Backup    bool
	Verify    bool
	Verbose   bool
	DryRun    bool
	Include   []string
	Exclude   []string
}

// PackOpts holds parsed pack command flags
type PackOpts struct {
	Dir     string
	Output  string
	PassEnv string
	Depth   int
	Include []string
	Exclude []string
	Verbose bool
	DryRun  bool
}

// ListOpts holds parsed list command flags
type ListOpts struct {
	Archive   string
	PassEnv   string
	All       bool
	Verbose   bool
	Sizes     bool
	Dates     bool
	Checksums bool
	Patterns  []string
	SortBy    string
	Reverse   bool
	Format    string
	Limit     int
}

// initApp checks initialization and creates app
func initApp() (*types.App, error) {
	if !config.IsInitialized() {
		return nil, fmt.Errorf("goingenv is not initialized in this directory. Run 'goingenv init' first")
	}
	return NewApp()
}

// getPass retrieves password with cleanup function
func getPass(envVar string) (key string, cleanup func(), err error) {
	opts := password.Options{PasswordEnv: envVar}
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

// confirm prompts user for y/N confirmation
func confirm(prompt string) bool {
	if !term.IsTerminal(syscall.Stdin) {
		return true // non-interactive mode
	}
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	_, _ = fmt.Scanln(&response) //nolint:errcheck // user input may be empty
	return response == "y" || response == "Y" || response == "yes"
}

// pickArchive selects archive file or returns most recent
func pickArchive(app *types.App, specified string) (string, error) {
	if specified != "" {
		return specified, nil
	}

	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return "", fmt.Errorf("failed to find archives: %w", err)
	}
	if len(archives) == 0 {
		return "", fmt.Errorf("no archives found in %s directory. Use -f flag to specify an archive", config.GetGoingEnvDir())
	}
	return archives[len(archives)-1], nil
}

// parseUnpackOpts parses unpack command flags
func parseUnpackOpts(cmd *cobra.Command) (*UnpackOpts, error) {
	o := &UnpackOpts{}
	var err error

	if o.Archive, err = cmd.Flags().GetString("file"); err != nil {
		return nil, fmt.Errorf("failed to get file flag: %w", err)
	}
	if o.Target, err = cmd.Flags().GetString("target"); err != nil {
		return nil, fmt.Errorf("failed to get target flag: %w", err)
	}
	if o.Target == "" {
		o.Target = "."
	}
	if o.PassEnv, err = cmd.Flags().GetString("password-env"); err != nil {
		return nil, fmt.Errorf("failed to get password-env flag: %w", err)
	}
	if o.Overwrite, err = cmd.Flags().GetBool("overwrite"); err != nil {
		return nil, fmt.Errorf("failed to get overwrite flag: %w", err)
	}
	if o.Backup, err = cmd.Flags().GetBool("backup"); err != nil {
		return nil, fmt.Errorf("failed to get backup flag: %w", err)
	}
	if o.Verify, err = cmd.Flags().GetBool("verify"); err != nil {
		return nil, fmt.Errorf("failed to get verify flag: %w", err)
	}
	if o.Verbose, err = cmd.Flags().GetBool("verbose"); err != nil {
		return nil, fmt.Errorf("failed to get verbose flag: %w", err)
	}
	if o.DryRun, err = cmd.Flags().GetBool("dry-run"); err != nil {
		return nil, fmt.Errorf("failed to get dry-run flag: %w", err)
	}
	if o.Include, err = cmd.Flags().GetStringSlice("include"); err != nil {
		return nil, fmt.Errorf("failed to get include flag: %w", err)
	}
	if o.Exclude, err = cmd.Flags().GetStringSlice("exclude"); err != nil {
		return nil, fmt.Errorf("failed to get exclude flag: %w", err)
	}

	return o, nil
}

// parsePackOpts parses pack command flags
func parsePackOpts(cmd *cobra.Command) (*PackOpts, error) {
	o := &PackOpts{}
	var err error

	if o.Dir, err = cmd.Flags().GetString("directory"); err != nil {
		return nil, fmt.Errorf("failed to get directory flag: %w", err)
	}
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.Output, err = cmd.Flags().GetString("output"); err != nil {
		return nil, fmt.Errorf("failed to get output flag: %w", err)
	}
	if o.Output == "" {
		o.Output = config.GetDefaultArchivePath()
	} else if !filepath.IsAbs(o.Output) {
		o.Output = filepath.Join(config.GetGoingEnvDir(), o.Output)
	}
	if o.PassEnv, err = cmd.Flags().GetString("password-env"); err != nil {
		return nil, fmt.Errorf("failed to get password-env flag: %w", err)
	}
	if o.Depth, err = cmd.Flags().GetInt("depth"); err != nil {
		return nil, fmt.Errorf("failed to get depth flag: %w", err)
	}
	if o.Include, err = cmd.Flags().GetStringSlice("include"); err != nil {
		return nil, fmt.Errorf("failed to get include flag: %w", err)
	}
	if o.Exclude, err = cmd.Flags().GetStringSlice("exclude"); err != nil {
		return nil, fmt.Errorf("failed to get exclude flag: %w", err)
	}
	if o.Verbose, err = cmd.Flags().GetBool("verbose"); err != nil {
		return nil, fmt.Errorf("failed to get verbose flag: %w", err)
	}
	if o.DryRun, err = cmd.Flags().GetBool("dry-run"); err != nil {
		return nil, fmt.Errorf("failed to get dry-run flag: %w", err)
	}

	return o, nil
}

// parseListOpts parses list command flags
func parseListOpts(cmd *cobra.Command) (*ListOpts, error) {
	o := &ListOpts{}
	var err error

	if o.Archive, err = cmd.Flags().GetString("file"); err != nil {
		return nil, fmt.Errorf("failed to get file flag: %w", err)
	}
	if o.PassEnv, err = cmd.Flags().GetString("password-env"); err != nil {
		return nil, fmt.Errorf("failed to get password-env flag: %w", err)
	}
	if o.All, err = cmd.Flags().GetBool("all"); err != nil {
		return nil, fmt.Errorf("failed to get all flag: %w", err)
	}
	if o.Verbose, err = cmd.Flags().GetBool("verbose"); err != nil {
		return nil, fmt.Errorf("failed to get verbose flag: %w", err)
	}
	if o.Sizes, err = cmd.Flags().GetBool("sizes"); err != nil {
		return nil, fmt.Errorf("failed to get sizes flag: %w", err)
	}
	if o.Dates, err = cmd.Flags().GetBool("dates"); err != nil {
		return nil, fmt.Errorf("failed to get dates flag: %w", err)
	}
	if o.Checksums, err = cmd.Flags().GetBool("checksums"); err != nil {
		return nil, fmt.Errorf("failed to get checksums flag: %w", err)
	}
	if o.Patterns, err = cmd.Flags().GetStringSlice("pattern"); err != nil {
		return nil, fmt.Errorf("failed to get pattern flag: %w", err)
	}
	if o.SortBy, err = cmd.Flags().GetString("sort"); err != nil {
		return nil, fmt.Errorf("failed to get sort flag: %w", err)
	}
	if o.Reverse, err = cmd.Flags().GetBool("reverse"); err != nil {
		return nil, fmt.Errorf("failed to get reverse flag: %w", err)
	}
	if o.Format, err = cmd.Flags().GetString("format"); err != nil {
		return nil, fmt.Errorf("failed to get format flag: %w", err)
	}
	if o.Limit, err = cmd.Flags().GetInt("limit"); err != nil {
		return nil, fmt.Errorf("failed to get limit flag: %w", err)
	}

	return o, nil
}

// buildScanOpts creates ScanOptions from PackOpts and config
func buildScanOpts(p *PackOpts, cfg *types.Config) *types.ScanOptions {
	opts := &types.ScanOptions{
		RootPath:        p.Dir,
		MaxDepth:        p.Depth,
		Patterns:        p.Include,
		ExcludePatterns: p.Exclude,
	}

	if opts.MaxDepth == 0 {
		opts.MaxDepth = cfg.DefaultDepth
	}
	if len(opts.Patterns) == 0 {
		opts.Patterns = cfg.EnvPatterns
	}
	if len(opts.ExcludePatterns) == 0 {
		opts.ExcludePatterns = cfg.ExcludePatterns
	} else {
		opts.ExcludePatterns = append(opts.ExcludePatterns, cfg.ExcludePatterns...)
	}

	return opts
}
