package main

import (
	"fmt"
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
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
