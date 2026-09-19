package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"goingenv/pkg/envfile"
	"goingenv/pkg/types"
)

// RunOpts holds parsed run command flags.
type RunOpts struct {
	PassOpts
	Env   string
	Files []string // archive entries to load; empty means the root .env
}

// newRunCommand creates the run command
func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] -- COMMAND [ARGS...]",
		Short: "Run a command with an archive's variables in its environment",
		Long: `Decrypt an archive in memory and run COMMAND with its variables set.
Nothing is written to disk.

By default the archive's root .env is loaded. --file selects other entries by
their path inside the archive and may be repeated; a later file wins on a
duplicate key. Archive values override variables already in the environment.

Flag parsing stops at COMMAND, so its own flags pass through untouched. The
command's exit status becomes goingenv's; 127 means COMMAND was not found.

Examples:
  goingenv run -- npm start
  goingenv run --env prod -- ./deploy.sh --region eu
  goingenv run --file apps/api/.env --file .env.local -- go test ./...
  printf 'pw\n' | goingenv run --password-stdin -- sh -c 'echo $DATABASE_URL'`,
		Args: cobra.MinimumNArgs(1),
		RunE: runRunCommand,
	}
	cmd.Flags().SetInterspersed(false)

	addPasswordFlags(cmd)
	addEnvFlag(cmd)
	cmd.Flags().StringSlice("file", nil, "Archive entry to load (repeatable; default: .env)")

	return cmd
}

func parseRunOpts(cmd *cobra.Command) (*RunOpts, error) {
	o := &RunOpts{}
	var err error
	if o.PassOpts, err = parsePassOpts(cmd); err != nil {
		return nil, err
	}
	if o.Env, err = parseEnvFlag(cmd); err != nil {
		return nil, err
	}
	if o.Files, err = cmd.Flags().GetStringSlice("file"); err != nil {
		return nil, fmt.Errorf("failed to get file flag: %w", err)
	}
	if len(o.Files) == 0 {
		o.Files = []string{".env"}
	}
	return o, nil
}

// runRunCommand executes the run command. It writes nothing to stdout
// itself: that stream belongs to the child.
func runRunCommand(cmd *cobra.Command, args []string) error {
	if _, err := outputFor(cmd, false); err != nil {
		return err
	}

	app, err := initApp()
	if err != nil {
		return err
	}

	opts, err := parseRunOpts(cmd)
	if err != nil {
		return err
	}

	archivePath, err := pickArchive(app, "", opts.Env)
	if err != nil {
		return err
	}

	key, cleanup, err := getPass(opts.PassOpts, opts.Env, false)
	if err != nil {
		return err
	}
	defer cleanup()

	vars, err := loadArchiveVars(app, archivePath, key, opts.Files)
	if err != nil {
		return err
	}

	return execWithEnv(args, vars)
}

// loadArchiveVars decrypts the archive and merges the requested entries in
// order, later files winning.
func loadArchiveVars(app *types.App, archivePath, key string, entries []string) (map[string]string, error) {
	_, files, err := app.Archiver.ReadFiles(archivePath, key)
	if err != nil {
		return nil, describeDecryptError(err)
	}

	vars := make(map[string]string)
	for _, name := range entries {
		data, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("%s is not in the archive; it contains: %s", name, strings.Join(sortedKeys(files), ", "))
		}
		parsed, parseErr := envfile.Parse(data)
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", name, parseErr)
		}
		for k, v := range parsed.Map() {
			vars[k] = v
		}
	}
	return vars, nil
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// execWithEnv runs argv with vars layered over the current environment and
// returns an ExitError carrying the child's status. Signals are forwarded so
// a Ctrl-C or a supervisor's SIGTERM reaches the child.
func execWithEnv(argv []string, vars map[string]string) error {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return &ExitError{Code: 127, Err: fmt.Errorf("command not found: %s", argv[0])}
	}

	child := exec.Command(path, argv[1:]...)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	// os/exec keeps the last occurrence of a duplicated key, so appending
	// after the inherited environment makes the archive win.
	child.Env = os.Environ()
	for _, k := range sortedVarNames(vars) {
		child.Env = append(child.Env, k+"="+vars[k])
	}

	if startErr := child.Start(); startErr != nil {
		return fmt.Errorf("failed to start %s: %w", argv[0], startErr)
	}

	signals := make(chan os.Signal, 8)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	defer signal.Stop(signals)
	go func() {
		for sig := range signals {
			_ = child.Process.Signal(sig) //nolint:errcheck // the child may already have exited
		}
	}()

	err = child.Wait()
	close(signals)
	return childExit(err)
}

func sortedVarNames(vars map[string]string) []string {
	names := make([]string, 0, len(vars))
	for k := range vars {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// childExit maps the child's termination to goingenv's exit status: its own
// code, or 128+signal when a signal killed it, following the shell convention.
func childExit(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return fmt.Errorf("failed to run command: %w", err)
	}
	code := exitErr.ExitCode()
	if code == -1 {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			code = 128 + int(status.Signal())
		} else {
			code = 1
		}
	}
	return &ExitError{Code: code}
}
