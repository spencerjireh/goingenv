package main

import (
	"fmt"
	"os"

	"goingenv/internal/cli"
)

// Version information - can be set during build
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
)

func main() {
	// Initialize and execute the root command
	rootCmd := cli.NewRootCommand(Version)
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}