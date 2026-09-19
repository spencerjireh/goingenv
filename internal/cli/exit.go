package cli

import "errors"

// ExitError carries a specific process exit code out of a command. A nil Err
// means the command already said everything it had to (or, for run, the
// child process did), so ReportError prints nothing.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode maps a command's returned error to the process exit status:
// nil is 0, an ExitError carries its own code, anything else is 1.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return 1
}
