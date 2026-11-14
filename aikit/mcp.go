package aikit

import (
	"context"
	"errors"
	"os"

	"github.com/firebase/genkit/go/genkit"
	genkitmcp "github.com/firebase/genkit/go/plugins/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// MCPServer exposes Genkit tools through the Model Context Protocol.
type MCPServer struct {
	*genkitmcp.GenkitMCPServer
}

// NewMCPServer creates an MCP server with the supplied tools.
func NewMCPServer(ctx context.Context, serverName string, tools []Tool) *MCPServer {
	g := genkit.Init(ctx)

	for _, tool := range tools {
		genkit.RegisterAction(g, tool.Ref)
	}

	return &MCPServer{
		GenkitMCPServer: genkitmcp.NewMCPServer(
			g,
			genkitmcp.MCPServerOptions{
				Name: serverName,
			}),
	}
}

// Start serves MCP requests over standard input and output until stopped.
func (s *MCPServer) Start(ctx context.Context) error {
	if err := mcpserver.NewStdioServer(s.GetServer()).Listen(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(
		err,
		context.Canceled,
	) {
		return err
	}

	return nil
}
