// ABOUTME: Root command for memo CLI with config-driven storage initialization.
// ABOUTME: Handles global flags and persistent pre/post run hooks.

package main

import (
	"fmt"

	"github.com/harperreed/memo/internal/config"
	"github.com/harperreed/memo/internal/storage"
	"github.com/spf13/cobra"
)

const banner = `
memo - Markdown notes
`

var (
	store storage.Storage
)

var rootCmd = &cobra.Command{
	Use:   "memo",
	Short: "A CLI notes tool with markdown support",
	Long:  banner + `memo is a command-line notes tool that stores markdown notes with tags and attachments.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip storage init for version and migrate commands
		if cmd.Name() == "version" || cmd.Name() == "migrate" {
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		store, err = cfg.OpenStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if store != nil {
			return store.Close()
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
