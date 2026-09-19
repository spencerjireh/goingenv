package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
	"goingenv/internal/constants"
	"goingenv/pkg/password"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// newListCommand creates the list command
func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List archive contents",
		Long: `Display the contents of an encrypted archive without extracting files.

The list command:
- Decrypts the archive metadata using the password you supply
- Shows the archive's creation date, version and description
- Lists the files it contains with their sizes and timestamps
- Optionally filters those files by pattern

Examples:
  goingenv list                                         # Newest unnamed archive, prompt for password
  goingenv list --env prod                              # Newest prod-<timestamp>.enc
  goingenv list -f backup.enc                           # A specific archive
  goingenv list --password-env MY_PASSWORD --all        # List all archives with env password
  goingenv list --all --env prod                        # Only the prod archives
  goingenv list -f archive.enc --pattern "*.env.prod*"  # Filter files by pattern`,
		RunE: runListCommand,
	}

	addPasswordFlags(cmd)
	addEnvFlag(cmd)
	cmd.Flags().StringP("file", "f", "", "List this archive (default: most recent for the environment)")
	cmd.Flags().Bool("all", false, "List contents of all available archives")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed file information")
	cmd.Flags().Bool("sizes", false, "Show detailed output")
	cmd.Flags().Bool("dates", false, "Show file modification dates")
	cmd.Flags().Bool("checksums", false, "Show file checksums")
	cmd.Flags().StringSliceP("pattern", "p", nil, "Filter files by patterns (glob-style)")
	cmd.Flags().StringP("sort", "s", "name", "Sort files by: name, size, date, type")
	cmd.Flags().Bool("reverse", false, "Reverse sort order")
	cmd.Flags().IntP("limit", "l", 0, "Limit number of files to show (0 = no limit)")

	return cmd
}

// runListCommand executes the list command
func runListCommand(cmd *cobra.Command, args []string) error {
	// csv is accepted here and nowhere else: this command supported it before
	// --format was global, and dropping it would break existing callers.
	out, err := outputFor(cmd, true)
	if err != nil {
		return err
	}

	app, err := initApp()
	if err != nil {
		out.Header()
		out.Blank()
		return err
	}

	opts, err := parseListOpts(cmd)
	if err != nil {
		return err
	}

	if opts.All {
		return listAllArchives(out, app, opts)
	}

	if opts.Archive, err = pickArchive(app, opts.Archive, opts.Env); err != nil {
		return err
	}

	if _, statErr := os.Stat(opts.Archive); os.IsNotExist(statErr) {
		return fmt.Errorf("archive not found: %s", opts.Archive)
	}

	key, cleanup, err := getPass(opts.PassOpts, opts.Env, false)
	if err != nil {
		return err
	}
	defer cleanup()

	out.Header()
	out.Blank()

	archive, err := app.Archiver.List(opts.Archive, key)
	if err != nil {
		return describeDecryptError(err)
	}

	// Archive info
	out.Section(filepath.Base(opts.Archive))
	out.Indent(fmt.Sprintf("Created: %s", archive.CreatedAt.Format(constants.DateTimeFormat)))
	out.Indent(fmt.Sprintf("Version: %s", archive.Version))
	if archive.Env != "" {
		out.Indent(fmt.Sprintf("Environment: %s", archive.Env))
	}
	out.Blank()

	filesToShow := archive.Files
	if len(opts.Patterns) > 0 {
		filesToShow = filterFilesByPatterns(archive.Files, opts.Patterns)
		out.MutedPrint(fmt.Sprintf("  Showing %d files matching patterns (out of %d total)",
			len(filesToShow), len(archive.Files)))
		out.Blank()
	}

	sortFiles(filesToShow, opts.SortBy, opts.Reverse)

	if opts.Limit > 0 && len(filesToShow) > opts.Limit {
		filesToShow = filesToShow[:opts.Limit]
	}

	// The header and archive summary above went to stderr in a machine format,
	// so stdout carries only what follows.
	if handled, emitErr := emitListResult(out, archive, filesToShow, opts); handled {
		return emitErr
	}

	displayFilesTable(out, filesToShow, opts)

	// Summary
	out.Blank()
	var totalSize int64
	for _, f := range filesToShow {
		totalSize += f.Size
	}
	out.MutedPrint(fmt.Sprintf("  %d files, %s total", len(filesToShow), utils.FormatSize(totalSize)))

	return nil
}

// listAllArchives lists every archive, or every archive for --env.
//
// With --verbose the password is resolved once, up front, from the
// non-interactive sources only: stdin can be read a single time, and a prompt
// per archive would be hostile. Without one the listing degrades to names
// and sizes. Without --verbose nothing is decrypted, so no password is
// resolved: an unset --password-env or an empty stdin must not fail a plain
// listing.
func listAllArchives(out *Output, app *types.App, opts *ListOpts) error {
	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		out.Header()
		out.Blank()
		return fmt.Errorf("failed to find archives: %w", err)
	}
	if opts.Env != "" {
		archives = archivesForEnv(archives, opts.Env)
	}

	var key string
	var haveKey bool
	if opts.Verbose {
		if key, haveKey, err = password.ResolveNonInteractive(passwordOptionsFor(opts.PassOpts, opts.Env, false)); err != nil {
			return err
		}
		defer password.ClearPassword(&key)
	}

	out.Header()
	out.Blank()

	if len(archives) == 0 {
		out.Warning("No archives found")
		out.Hint(fmt.Sprintf("Archives are read from the %s directory", config.GetGoingEnvDir()))
		return nil
	}

	out.Section(fmt.Sprintf("Archives (%d)", len(archives)))
	out.Blank()

	for i, archivePath := range archives {
		name := filepath.Base(archivePath)
		info, statErr := os.Stat(archivePath)
		if statErr != nil {
			continue
		}

		out.Printf("  [%d] %s\n", i+1, name)
		out.Indent(fmt.Sprintf("    Size: %s", utils.FormatSize(info.Size())))
		out.Indent(fmt.Sprintf("    Modified: %s", info.ModTime().Format(constants.DateTimeFormat)))

		if opts.Verbose && haveKey {
			describeArchive(out, app, archivePath, key)
		}

		out.Blank()
	}

	if opts.Verbose && !haveKey {
		out.Hint("Provide a password with --password-env or --password-stdin to see detailed archive information")
	}

	return nil
}

// describeArchive adds the decrypted summary lines for one archive in the
// --all --verbose listing.
func describeArchive(out *Output, app *types.App, archivePath, key string) {
	archive, listErr := app.Archiver.List(archivePath, key)
	if listErr != nil {
		out.Indent(fmt.Sprintf("    Status: cannot read (%v)", describeDecryptError(listErr)))
		return
	}
	out.Indent(fmt.Sprintf("    Files: %d", len(archive.Files)))
	out.Indent(fmt.Sprintf("    Total size: %s", utils.FormatSize(archive.TotalSize)))
	if archive.Env != "" {
		out.Indent(fmt.Sprintf("    Environment: %s", archive.Env))
	}
}

// displayFilesTable displays files in table format
func displayFilesTable(out *Output, files []types.EnvFile, opts *ListOpts) {
	if len(files) == 0 {
		out.MutedPrint("  No files to display")
		return
	}

	out.Section("Files")

	for _, file := range files {
		var parts []string
		parts = append(parts, file.RelativePath)

		if opts.Sizes || opts.Verbose {
			parts = append(parts, utils.FormatSize(file.Size))
		}
		if opts.Dates || opts.Verbose {
			parts = append(parts, file.ModTime.Format(constants.DateTimeFormat))
		}
		if opts.Checksums {
			cs := file.Checksum
			if len(cs) > 16 {
				cs = cs[:16] + "..."
			}
			parts = append(parts, cs)
		}

		out.Indent(strings.Join(parts, "    "))
	}
}

// displayFilesCSV displays files in CSV format
func displayFilesCSV(out *Output, files []types.EnvFile) {
	fmt.Fprintln(out.stdout, "name,path,size,modified,checksum")
	for _, file := range files {
		fmt.Fprintf(out.stdout, "%s,%s,%d,%s,%s\n",
			filepath.Base(file.RelativePath),
			file.RelativePath,
			file.Size,
			file.ModTime.Format(constants.DateTimeFormat),
			file.Checksum)
	}
}

// filterFilesByPatterns filters files based on glob patterns
func filterFilesByPatterns(files []types.EnvFile, patterns []string) []types.EnvFile {
	var filtered []types.EnvFile

	for _, file := range files {
		for _, pattern := range patterns {
			matched, matchErr := filepath.Match(pattern, file.RelativePath)
			if matchErr == nil && matched {
				filtered = append(filtered, file)
				break
			}
		}
	}

	return filtered
}

// sortFiles sorts files based on the specified criteria
func sortFiles(files []types.EnvFile, sortBy string, reverse bool) {
	switch sortBy {
	case "size":
		sort.Slice(files, func(i, j int) bool {
			if reverse {
				return files[i].Size > files[j].Size
			}
			return files[i].Size < files[j].Size
		})
	case "date":
		sort.Slice(files, func(i, j int) bool {
			if reverse {
				return files[i].ModTime.After(files[j].ModTime)
			}
			return files[i].ModTime.Before(files[j].ModTime)
		})
	case "type":
		sort.Slice(files, func(i, j int) bool {
			ext1 := filepath.Ext(files[i].RelativePath)
			ext2 := filepath.Ext(files[j].RelativePath)
			if reverse {
				return ext1 > ext2
			}
			return ext1 < ext2
		})
	default: // name
		sort.Slice(files, func(i, j int) bool {
			if reverse {
				return files[i].RelativePath > files[j].RelativePath
			}
			return files[i].RelativePath < files[j].RelativePath
		})
	}
}

// emitListResult writes the machine-readable and csv renderings. It reports
// whether it handled the format, so the caller falls through to the human
// table for text without this function having to know how that is drawn.
func emitListResult(out *Output, archive *types.Archive, files []types.EnvFile, opts *ListOpts) (bool, error) {
	switch out.Format() {
	case FormatJSON:
		return true, out.EmitJSON(ListPayload{
			Archive: ListArchiveInfo{
				Name:        filepath.Base(opts.Archive),
				Path:        opts.Archive,
				CreatedAt:   archive.CreatedAt,
				Version:     archive.Version,
				Description: archive.Description,
				Env:         archive.Env,
			},
			Files: toFileInfos(files),
			Count: len(files),
		})
	case FormatPorcelain:
		records := make([][]string, 0, len(files))
		for _, f := range files {
			records = append(records, []string{
				f.RelativePath,
				strconv.FormatInt(f.Size, 10),
				f.ModTime.UTC().Format(time.RFC3339),
				f.Checksum,
			})
		}
		return true, out.EmitPorcelain(records)
	case FormatCSV:
		displayFilesCSV(out, files)
		return true, nil
	default:
		return false, nil
	}
}
