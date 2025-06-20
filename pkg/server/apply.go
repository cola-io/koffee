package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// ApplyResource returns a function that applies a resource.
func (s *Server) ApplyResource() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		manifest, err := req.RequireString("manifest")
		if err != nil {
			return nil, err
		}

		// TODO: Implement resource application logic

		return mcp.NewToolResultText(manifest), nil
	}
}
