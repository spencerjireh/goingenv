package cli

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"plain error", errors.New("x"), 1},
		{"exit error", &ExitError{Code: 7}, 7},
		{"wrapped exit error", fmt.Errorf("ctx: %w", &ExitError{Code: 127, Err: errors.New("nf")}), 127},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Errorf("ExitCode = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestNewDiffCommand(t *testing.T) {
	cmd := newDiffCommand()
	for _, flag := range []string{"password-env", "password-stdin", "env", "exit-code"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("diff command missing --%s flag", flag)
		}
	}
}

func TestNewRunCommand(t *testing.T) {
	cmd := newRunCommand()
	for _, flag := range []string{"password-env", "password-stdin", "env", "file"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("run command missing --%s flag", flag)
		}
	}
	// Flags after the command belong to the command, not to goingenv.
	if err := cmd.ParseFlags([]string{"--env", "prod", "npm", "start", "--port", "3000"}); err != nil {
		t.Fatal(err)
	}
	if got := cmd.Flags().Args(); len(got) != 4 || got[0] != "npm" || got[2] != "--port" {
		t.Errorf("args = %v, want the child's argv intact", got)
	}
}
