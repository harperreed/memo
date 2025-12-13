// ABOUTME: Entry point for memo CLI application.
// ABOUTME: Initializes and executes the root command.

package main

import (
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
