package cli_test

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"goingenv/test/testutils"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDiff_WorktreeAndArchives(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password

	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "-o", "before.enc"))

	// Nothing changed yet.
	same := testutils.RunCLIWithPassword(t, dir, pw, "diff", "before.enc", "--exit-code")
	testutils.AssertSuccess(t, same)
	testutils.AssertOutputContains(t, same, "No differences")

	// Change a value, add a key, delete a file.
	writeFile(t, filepath.Join(dir, ".env"), "DATABASE_URL=postgres://localhost/test\nAPI_KEY=rotated\nSECRET_KEY=mysecret\nNEW_KEY=1")
	if err := os.Remove(filepath.Join(dir, "nested", "deep", ".env")); err != nil {
		t.Fatal(err)
	}

	porcelain := testutils.RunCLIWithPassword(t, dir, pw, "diff", "before.enc", "--format", "porcelain")
	testutils.AssertSuccess(t, porcelain)
	assertCleanStdout(t, porcelain)
	want := []string{
		"key\t.env\tAPI_KEY\tchanged",
		"key\t.env\tNEW_KEY\tadded",
		"file\tnested/deep/.env\tremoved",
		"key\tnested/deep/.env\tDEEP_CONFIG\tremoved",
		"key\tnested/deep/.env\tNESTED_VAR\tremoved",
	}
	got := nonEmptyLines(porcelain.Stdout)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("porcelain rows:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// Values never appear in any format.
	for _, format := range []string{"text", "json", "porcelain"} {
		r := testutils.RunCLIWithPassword(t, dir, pw, "diff", "before.enc", "--format", format)
		testutils.AssertSuccess(t, r)
		for _, leak := range []string{"rotated", "test123", "deep_value"} {
			if strings.Contains(r.Combined(), leak) {
				t.Errorf("%s output leaks %q", format, leak)
			}
		}
	}

	// --exit-code reports differences with status 1 and prints nothing extra.
	exit := testutils.RunCLIWithPassword(t, dir, pw, "diff", "before.enc", "--exit-code", "--format", "json")
	testutils.AssertExitCode(t, exit, 1)
	payload := decodeStdout(t, exit)
	if payload["changed"] != true {
		t.Errorf("changed = %v", payload["changed"])
	}
	if to, ok := payload["to"].(map[string]any); !ok || to["kind"] != "worktree" {
		t.Errorf("to = %v", payload["to"])
	}

	// Archive vs archive, with the default FROM being the newest unnamed one.
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack"))
	two := testutils.RunCLIWithPassword(t, dir, pw, "diff", "before.enc", "--format", "json")
	testutils.AssertSuccess(t, two)
	if decodeStdout(t, two)["changed"] != true {
		t.Error("before.enc vs worktree should differ")
	}
	latestVsTree := testutils.RunCLIWithPassword(t, dir, pw, "diff", "--format", "json")
	testutils.AssertSuccess(t, latestVsTree)
	if decodeStdout(t, latestVsTree)["changed"] != false {
		t.Error("newest archive was just packed from this tree; expected no differences")
	}
	ab := testutils.RunCLIWithPassword(t, dir, pw, "diff", "before.enc", findArchive(t, dir), "--format", "json")
	testutils.AssertSuccess(t, ab)
	if from, ok := decodeStdout(t, ab)["from"].(map[string]any); !ok || from["kind"] != "archive" {
		t.Errorf("from = %v", from)
	}
}

func TestRun_InjectsVariables(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack"))

	// Delete the plaintext so only the archive can supply the values.
	for _, f := range []string{".env", ".env.production"} {
		if err := os.Remove(filepath.Join(dir, f)); err != nil {
			t.Fatal(err)
		}
	}

	result := testutils.RunCLIWithPassword(t, dir, pw, "run", "--", "sh", "-c", "echo $API_KEY")
	testutils.AssertSuccess(t, result)
	if strings.TrimSpace(result.Stdout) != "test123" {
		t.Errorf("stdout = %q, want only the child's output", result.Stdout)
	}
	testutils.AssertFileNotExists(t, filepath.Join(dir, ".env"))

	// Archive values override the host environment.
	override := testutils.RunCLIWithEnv(t, dir, map[string]string{"GOINGENV_PASSWORD": pw, "API_KEY": "from-host"},
		"run", "--password-env", "GOINGENV_PASSWORD", "--", "sh", "-c", "echo $API_KEY")
	testutils.AssertSuccess(t, override)
	if strings.TrimSpace(override.Stdout) != "test123" {
		t.Errorf("archive should override host: %q", override.Stdout)
	}

	// --file selects entries; later wins on duplicates; child flags pass through.
	multi := testutils.RunCLIWithPassword(t, dir, pw, "run", "--file", ".env.development", "--file", ".env.production",
		"--", "sh", "-c", "echo $NODE_ENV $DB_HOST")
	testutils.AssertSuccess(t, multi)
	if strings.TrimSpace(multi.Stdout) != "production prod.example.com" {
		t.Errorf("multi-file = %q", multi.Stdout)
	}

	missing := testutils.RunCLIWithPassword(t, dir, pw, "run", "--file", "nope/.env", "--", "true")
	testutils.AssertFailure(t, missing)
	testutils.AssertStderrContains(t, missing, "nope/.env")
}

func TestRun_ExitCodesAndStdin(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack"))

	seven := testutils.RunCLIWithPassword(t, dir, pw, "run", "--", "sh", "-c", "exit 7")
	testutils.AssertExitCode(t, seven, 7)
	if strings.Contains(seven.Stderr, "error") {
		t.Errorf("a child's exit status must not be reported as a goingenv error:\n%s", seven.Stderr)
	}

	notFound := testutils.RunCLIWithPassword(t, dir, pw, "run", "--", "goingenv-no-such-command-xyz")
	testutils.AssertExitCode(t, notFound, 127)
	testutils.AssertStderrContains(t, notFound, "not found")

	// The password is the first stdin line; the rest belongs to the child.
	passthrough := testutils.RunCLIWithStdin(t, dir, pw+"\nhello from stdin\n", nil,
		"run", "--password-stdin", "--", "sh", "-c", "cat")
	testutils.AssertSuccess(t, passthrough)
	if strings.TrimSpace(passthrough.Stdout) != "hello from stdin" {
		t.Errorf("child stdin = %q", passthrough.Stdout)
	}
}

// runAndSignal starts `goingenv run -- sh -c script`, waits for the script to
// print "ready", sends sig to goingenv alone, and returns the rest of the
// child's stdout with goingenv's exit code.
func runAndSignal(t *testing.T, dir, pw string, sig syscall.Signal, script string) (stdout string, code int) {
	t.Helper()
	cmd := testutils.CLICommand(t, dir, map[string]string{"GOINGENV_PASSWORD": pw}, "run", "--", "sh", "-c", script)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(pipe)
	ready, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(ready) != "ready" {
		t.Fatalf("waiting for the child: line %q, err %v", ready, err)
	}
	if err = cmd.Process.Signal(sig); err != nil {
		t.Fatal(err)
	}

	rest, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-time.After(15 * time.Second):
		if killErr := cmd.Process.Kill(); killErr != nil {
			t.Log(killErr)
		}
		t.Fatal("goingenv did not exit after the signal")
	}
	if err == nil {
		return string(rest), 0
	}
	return string(rest), cmd.ProcessState.ExitCode()
}

// A signal sent to goingenv alone reaches the child; a terminal signal does
// not get relayed, because the terminal already delivered it to the child.
func TestRun_SignalForwarding(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack"))

	// The shell runs a trap only once the foreground command returns, so the
	// child sleeps in short steps. A background `sleep & wait` would react
	// faster but can leak the sleep, which then holds the stdout pipe open.
	out, code := runAndSignal(t, dir, pw, syscall.SIGTERM,
		`trap 'echo got TERM; exit 0' TERM; echo ready; while :; do sleep 0.2; done`)
	if code != 0 || !strings.Contains(out, "got TERM") {
		t.Errorf("SIGTERM: exit %d, stdout %q; want the child to see the signal", code, out)
	}

	// SIGINT to goingenv only: the parent must survive and must not pass it
	// on, so the child runs to completion untouched.
	out, code = runAndSignal(t, dir, pw, syscall.SIGINT,
		`trap 'echo got INT' INT; echo ready; sleep 1; echo done`)
	if code != 0 || strings.Contains(out, "got INT") || !strings.Contains(out, "done") {
		t.Errorf("SIGINT: exit %d, stdout %q; want the child untouched and goingenv alive", code, out)
	}
}
