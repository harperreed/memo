// ABOUTME: Root command for memo CLI with Charm KV initialization.
// ABOUTME: Handles global flags and persistent pre/post run hooks.

package main

import (
	"fmt"

	"github.com/harper/memo/internal/charm"
	"github.com/spf13/cobra"
)

const banner = `
███╗   ███╗███████╗███╗   ███╗ ██████╗
████╗ ████║██╔════╝████╗ ████║██╔═══██╗
██╔████╔██║█████╗  ██╔████╔██║██║   ██║
██║╚██╔╝██║██╔══╝  ██║╚██╔╝██║██║   ██║
██║ ╚═╝ ██║███████╗██║ ╚═╝ ██║╚██████╔╝
╚═╝     ╚═╝╚══════╝╚═╝     ╚═╝ ╚═════╝

         📝 Markdown notes with Charm ✨
`

var (
	charmClient *charm.Client
)

var rootCmd = &cobra.Command{
	Use:   "memo",
	Short: "A CLI notes tool with markdown support",
	Long:  banner + `memo is a command-line notes tool that stores markdown notes with tags and attachments using Charm KV.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip client init for version command
		if cmd.Name() == "version" {
			return nil
		}

		var err error
		charmClient, err = charm.GetClient()
		if err != nil {
			return fmt.Errorf("failed to initialize charm client: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Client is global and managed by charm package
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
