package cli

import (
	"fmt"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"goingenv/internal/config"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// UnpackOpts holds parsed unpack command flags
type UnpackOpts struct {
	PassOpts
	Archive   string
	Target    string
	Env       string
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
	PassOpts
	Dir        string
	Output     string
	Env        string
	Manifest   bool
	Depth      int
	Include    []string
	Exclude    []string
	EnvExclude []string
	Verbose    bool
	DryRun     bool
}

// ListOpts holds parsed list command flags
type ListOpts struct {
	PassOpts
	Archive   string
	Env       string
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
		return nil, fmt.Errorf("goingenv is not initialized in this directory: run 'goingenv init' first")
	}
	return NewApp()
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
	if o.PassOpts, err = parsePassOpts(cmd); err != nil {
		return nil, err
	}
	if o.Env, err = parseEnvFlag(cmd); err != nil {
		return nil, err
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

// parsePackOpts parses pack command flags. cfg supplies the manifest
// default, which --manifest / --no-manifest override.
func parsePackOpts(cmd *cobra.Command, cfg *types.Config) (*PackOpts, error) {
	o := &PackOpts{}
	var err error

	if o.Dir, err = cmd.Flags().GetString("directory"); err != nil {
		return nil, fmt.Errorf("failed to get directory flag: %w", err)
	}
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.PassOpts, err = parsePassOpts(cmd); err != nil {
		return nil, err
	}
	if o.Env, err = parseEnvFlag(cmd); err != nil {
		return nil, err
	}
	if o.Output, err = cmd.Flags().GetString("output"); err != nil {
		return nil, fmt.Errorf("failed to get output flag: %w", err)
	}
	if o.Output == "" {
		o.Output = config.GetArchivePath(o.Env)
	} else if !filepath.IsAbs(o.Output) {
		o.Output = filepath.Join(config.GetGoingEnvDir(), o.Output)
	}
	if o.Manifest, err = parseManifestFlags(cmd, cfg.Manifest); err != nil {
		return nil, err
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
	if o.EnvExclude, err = cmd.Flags().GetStringSlice("env-exclude"); err != nil {
		return nil, fmt.Errorf("failed to get env-exclude flag: %w", err)
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
	if o.PassOpts, err = parsePassOpts(cmd); err != nil {
		return nil, err
	}
	if o.Env, err = parseEnvFlag(cmd); err != nil {
		return nil, err
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

// parseManifestFlags resolves --manifest / --no-manifest over the config
// default. Passing both is contradictory and rejected.
func parseManifestFlags(cmd *cobra.Command, def bool) (bool, error) {
	on := cmd.Flags().Changed("manifest")
	off := cmd.Flags().Changed("no-manifest")
	switch {
	case on && off:
		return false, fmt.Errorf("--manifest and --no-manifest cannot be combined")
	case on:
		return true, nil
	case off:
		return false, nil
	default:
		return def, nil
	}
}

// buildScanOpts creates ScanOptions from PackOpts and config.
//
// The two exclude fields are unions of flag and config: both narrow the scan, so
// a flag must never be able to drop a configured exclusion. --include is the
// exception and replaces cfg.EnvPatterns, because a union of includes could
// never narrow to a single file.
//
// Every field is populated here rather than left empty, because Scanner.ScanFiles
// calls applyDefaults, which fills empty fields from the same config. Merging in
// both places would apply each config pattern twice.
func buildScanOpts(p *PackOpts, cfg *types.Config) *types.ScanOptions {
	opts := &types.ScanOptions{
		RootPath:           p.Dir,
		MaxDepth:           p.Depth,
		Patterns:           p.Include,
		ExcludePatterns:    mergePatterns(p.Exclude, cfg.ExcludePatterns),
		EnvExcludePatterns: mergePatterns(p.EnvExclude, cfg.EnvExcludePatterns),
	}

	if opts.MaxDepth == 0 {
		opts.MaxDepth = cfg.DefaultDepth
	}
	if len(opts.Patterns) == 0 {
		opts.Patterns = cfg.EnvPatterns
	}

	return opts
}

// mergePatterns returns the union of flag and config patterns in a fresh slice,
// so appending to the result can never write into the config's backing array.
func mergePatterns(flag, cfg []string) []string {
	merged := make([]string, 0, len(flag)+len(cfg))
	merged = append(merged, flag...)
	merged = append(merged, cfg...)
	return merged
}

// listFiles prints every file as a list item; verbose adds the size. Shared by
// pack and unpack so the two previews cannot drift apart.
func listFiles(out *Output, files []types.EnvFile, verbose bool) {
	for _, file := range files {
		if verbose {
			out.ListItem(fmt.Sprintf("%s (%s)", file.RelativePath, utils.FormatSize(file.Size)))
		} else {
			out.ListItem(file.RelativePath)
		}
	}
	out.Blank()
}

// outputFor builds the Output a command should use, honouring --format.
//
// Returns the error rather than a fallback Output: an unrecognised format has
// to fail loudly, since silently falling back to text would hand a script
// unparseable output while exiting 0.
func outputFor(cmd *cobra.Command, allowCSV bool) (*Output, error) {
	raw, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, fmt.Errorf("failed to get format flag: %w", err)
	}

	format, err := ParseFormat(raw, allowCSV)
	if err != nil {
		return nil, err
	}

	// Recorded so ReportError can render the failure in the same format the
	// command was going to use. Commands must not report errors themselves --
	// see ReportError for why.
	activeFormat = format

	return NewOutputFormat(appVersion, format), nil
}
