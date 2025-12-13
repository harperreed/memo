// ABOUTME: MCP server for memo integration with AI agents.
// ABOUTME: Provides tools, resources, and prompts for note management.

package mcp

import (
	"context"
	"database/sql"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	server *mcp.Server
	db     *sql.DB
}

func NewServer(db *sql.DB) *Server {
	s := &Server{db: db}

	s.server = mcp.NewServer(
		&mcp.Implementation{
			Name:    "memo",
			Version: "1.0.0",
		},
		&mcp.ServerOptions{
			HasTools:     true,
			HasResources: true,
			HasPrompts:   true,
		},
	)

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

func (s *Server) Serve(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}
