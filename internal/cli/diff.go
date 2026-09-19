package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"goingenv/internal/diff"
	"goingenv/pkg/types"
)

// DiffOpts holds parsed diff command flags and arguments.
type DiffOpts struct {
	PassOpts
	Env      string
	From     string // archive path; "" means the newest for Env
	To       string // archive path; "" means the working tree
	ExitCode bool
}

// newDiffCommand creates the diff command
func newDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [FROM [TO]]",
		Short: "Show which keys differ between archives, or an archive and the working tree",
		Long: `Compare two sets of environment files by key. Values are compared but never
printed, so the output is safe to paste into a review.

The sides follow git: FROM defaults to the newest archive for --env, and TO
defaults to the env files on disk (what pack would take). So:

  goingenv diff                 # newest archive -> working tree: what would pack change?
  goingenv diff old.enc         # old.enc -> working tree
  goingenv diff old.enc new.enc # old.enc -> new.enc

A bare file name is looked up under .goingenv. Both archives must open with
the same password.

Exit status is 0 whether or not anything differs; --exit-code makes it 1 when
there are differences, like git diff.`,
		Args: cobra.MaximumNArgs(2),
		RunE: runDiffCommand,
	}

	addPasswordFlags(cmd)
	addEnvFlag(cmd)
	cmd.Flags().Bool("exit-code", false, "Exit with 1 when there are differences, 0 when there are none")

	return cmd
}

func parseDiffOpts(cmd *cobra.Command, args []string) (*DiffOpts, error) {
	o := &DiffOpts{}
	var err error
	if o.PassOpts, err = parsePassOpts(cmd); err != nil {
		return nil, err
	}
	if o.Env, err = parseEnvFlag(cmd); err != nil {
		return nil, err
	}
	if o.ExitCode, err = cmd.Flags().GetBool("exit-code"); err != nil {
		return nil, fmt.Errorf("failed to get exit-code flag: %w", err)
	}
	if len(args) > 0 {
		o.From = resolveArchiveArg(args[0])
	}
	if len(args) > 1 {
		o.To = resolveArchiveArg(args[1])
	}
	return o, nil
}

// runDiffCommand executes the diff command
func runDiffCommand(cmd *cobra.Command, args []string) error {
	out, err := outputFor(cmd, false)
	if err != nil {
		return err
	}

	app, err := initApp()
	if err != nil {
		out.Header()
		out.Blank()
		return err
	}

	opts, err := parseDiffOpts(cmd, args)
	if err != nil {
		return err
	}

	if opts.From, err = pickArchive(app, opts.From, opts.Env); err != nil {
		return err
	}

	key, cleanup, err := getPass(opts.PassOpts, opts.Env, false)
	if err != nil {
		return err
	}
	defer cleanup()

	out.Header()
	out.Blank()

	from, err := sideFromArchive(app, opts.From, key)
	if err != nil {
		return err
	}
	to, toSide, err := toSideFor(app, opts, key)
	if err != nil {
		return err
	}

	result, err := diff.Compare(from, to)
	if err != nil {
		return fmt.Errorf("failed to compare: %w", err)
	}

	fromSide := DiffSide{Kind: "archive", Path: opts.From}
	if err := emitDiffResult(out, fromSide, toSide, result); err != nil {
		return err
	}
	if opts.ExitCode && result.Changed() {
		return &ExitError{Code: 1}
	}
	return nil
}

// toSideFor reads the TO side: a second archive when one was named, else the
// working tree.
func toSideFor(app *types.App, opts *DiffOpts, key string) (map[string][]byte, DiffSide, error) {
	if opts.To == "" {
		files, err := sideFromWorktree(app)
		return files, DiffSide{Kind: "worktree"}, err
	}
	files, err := sideFromArchive(app, opts.To, key)
	return files, DiffSide{Kind: "archive", Path: opts.To}, err
}

// sideFromArchive reads an archive's files into memory.
func sideFromArchive(app *types.App, path, key string) (map[string][]byte, error) {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return nil, fmt.Errorf("archive not found: %s", path)
	}
	_, files, err := app.Archiver.ReadFiles(path, key)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), describeDecryptError(err))
	}
	return files, nil
}

// sideFromWorktree scans the current directory exactly as pack would and
// reads every hit. An empty tree is a valid side: everything in the archive
// then reports as removed.
func sideFromWorktree(app *types.App) (map[string][]byte, error) {
	scanOpts := buildScanOpts(&PackOpts{Dir: "."}, app.Config)
	found, err := app.Scanner.ScanFiles(scanOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}
	files := make(map[string][]byte, len(found))
	for i := range found {
		data, readErr := os.ReadFile(found[i].Path)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read %s: %w", found[i].RelativePath, readErr)
		}
		files[found[i].RelativePath] = data
	}
	return files, nil
}

// emitDiffResult renders the result in the requested format.
func emitDiffResult(out *Output, from, to DiffSide, result diff.Result) error {
	switch out.Format() {
	case FormatJSON:
		return out.EmitJSON(DiffPayload{From: from, To: to, Files: toFileDiffs(result), Changed: result.Changed()})
	case FormatPorcelain:
		var records [][]string
		for _, f := range result.Files {
			if f.Status != diff.Modified {
				records = append(records, []string{porcelainKindFile, f.Path, f.Status})
			}
			for _, k := range f.Keys {
				records = append(records, []string{porcelainKindKey, f.Path, k.Key, k.Change})
			}
		}
		return out.EmitPorcelain(records)
	default:
		displayDiff(out, from, to, result)
		return nil
	}
}

func toFileDiffs(result diff.Result) []FileDiff {
	files := make([]FileDiff, 0, len(result.Files))
	for _, f := range result.Files {
		keys := make([]KeyDiff, 0, len(f.Keys))
		for _, k := range f.Keys {
			keys = append(keys, KeyDiff{Key: k.Key, Change: k.Change})
		}
		files = append(files, FileDiff{Path: f.Path, Status: f.Status, Keys: keys})
	}
	return files
}

func sideLabel(s DiffSide) string {
	if s.Kind == "worktree" {
		return "working tree"
	}
	return filepath.Base(s.Path)
}

func displayDiff(out *Output, from, to DiffSide, result diff.Result) {
	out.Action(fmt.Sprintf("%s -> %s", sideLabel(from), sideLabel(to)))
	out.Blank()
	if !result.Changed() {
		out.Success("No differences")
		return
	}
	for _, f := range result.Files {
		out.Section(fmt.Sprintf("%s %s (%s)", diff.Marker(f.Status), f.Path, f.Status))
		for _, k := range f.Keys {
			out.Indent(fmt.Sprintf("%s %s", diff.Marker(k.Change), k.Key))
		}
		out.Blank()
	}
	out.MutedPrint(fmt.Sprintf("  %d file(s) differ", len(result.Files)))
}
