package main

import (
	"os"

	"goingenv/internal/cli"
)

// Version information - can be set during build via -ldflags -X.
//
// GitCommit must exist for the linker to accept -X main.GitCommit: the build
// previously passed that flag with no corresponding variable, so the value was
// silently discarded.
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Initialize and execute the root command
	cli.SetBuildInfo(BuildTime, GitCommit)
	rootCmd := cli.NewRootCommand(Version)

	if err := rootCmd.Execute(); err != nil {
		// Reported through the CLI package so the message matches the
		// requested --format, and on stderr rather than the stdout this used
		// to use -- which corrupted every machine-readable format and any
		// `goingenv ... > file` redirect.
		cli.ReportError(err)
		os.Exit(1)
	}
}
