// ABOUTME: MCP command to start the MCP server.
// ABOUTME: Runs on stdio for integration with AI agents.

package main

import (
	"github.com/harper/memo/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server",
	Long:  `Start the Model Context Protocol server for AI agent integration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer(charmClient)
		return server.Serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
