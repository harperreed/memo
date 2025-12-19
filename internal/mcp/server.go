// ABOUTME: MCP server for memo integration with AI agents.
// ABOUTME: Provides tools, resources, and prompts for note management.

package mcp

import (
	"context"

	"github.com/harper/memo/internal/charm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	server *mcp.Server
	client *charm.Client
}

func NewServer(client *charm.Client) *Server {
	s := &Server{client: client}

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
