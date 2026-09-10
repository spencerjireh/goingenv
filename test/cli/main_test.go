package cli_test

import (
	"os"
	"testing"

	"goingenv/test/testutils"
)

// TestMain cleans up the binary that testutils.BuildBinary compiles and caches
// for this package, plus the shared fake home directory. Without it the
// package leaked a goingenv-binary-* temp dir on every run.
//
// CleanupBinary does not reset the sync.Once guards, so it must run after
// m.Run() has returned.
func TestMain(m *testing.M) {
	code := m.Run()
	testutils.CleanupBinary()
	os.Exit(code)
}
