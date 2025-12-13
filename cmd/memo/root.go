// ABOUTME: Root command for memo CLI with database initialization.
// ABOUTME: Handles global flags and persistent pre/post run hooks.

package main

import (
	"database/sql"
	"fmt"

	"github.com/harper/memo/internal/db"
	"github.com/spf13/cobra"
)

const banner = `
███╗   ███╗███████╗███╗   ███╗ ██████╗
████╗ ████║██╔════╝████╗ ████║██╔═══██╗
██╔████╔██║█████╗  ██╔████╔██║██║   ██║
██║╚██╔╝██║██╔══╝  ██║╚██╔╝██║██║   ██║
██║ ╚═╝ ██║███████╗██║ ╚═╝ ██║╚██████╔╝
╚═╝     ╚═╝╚══════╝╚═╝     ╚═╝ ╚═════╝

         📝 Markdown notes with MCP ✨
`

var (
	dbPath string
	dbConn *sql.DB
)

var rootCmd = &cobra.Command{
	Use:   "memo",
	Short: "A CLI notes tool with markdown support",
	Long:  banner + `memo is a command-line notes tool that stores markdown notes with tags and attachments in SQLite.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip DB init for version command
		if cmd.Name() == "version" {
			return nil
		}

		var err error
		if dbPath == "" {
			dbPath = db.DefaultPath()
		}
		dbConn, err = db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if dbConn != nil {
			return dbConn.Close()
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database path (default: ~/.local/share/memo/memo.db)")
}

func Execute() error {
	return rootCmd.Execute()
}
