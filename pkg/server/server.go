package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"cola.io/koffee/pkg/client"
	"cola.io/koffee/pkg/definition"
	"cola.io/koffee/pkg/mcp"
	"cola.io/koffee/pkg/version"
)

type ServerOption func(*Server)

type Server struct {
	svr       *server.MCPServer
	generator *definition.HumanReadableGenerator
	cb        client.ClientBuilder
	transport string
	port      int
}

// WithTransport sets the transport type for the server.
func WithTransport(t string) func(*Server) {
	return func(s *Server) {
		s.transport = t
	}
}

// WithPort sets the port for the server when the transport is sse.
func WithPort(p int) func(*Server) {
	return func(s *Server) {
		s.port = p
	}
}

// NewServer creates a new mcp server.
func NewServer(kubeconfig string, opts ...ServerOption) *Server {
	generator := definition.NewTableGenerator()
	definition.AddHandlers(generator)
	s := &Server{
		transport: "stdio",
		port:      8888,
		svr: server.NewMCPServer(
			"Kubernetes MCP Server",
			version.Get().Version,
			server.WithRecovery(),
			server.WithLogging(),
		),
		generator: generator,
		cb:        client.NewClientBuilder(kubeconfig),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RegisterTools registers the tools for the server.
func (s *Server) RegisterTools(ctx context.Context) {
	slog.Info("Registering tools")
	s.svr.AddTools([]server.ServerTool{
		{
			Tool:    mcp.MakeListClustersTool(),
			Handler: s.ListClusters(),
		},
		{
			Tool:    mcp.MakeSwitchContextTool(),
			Handler: s.SwitchContexts(),
		},
		{
			Tool:    mcp.MakeGetClusterVersionTool(),
			Handler: s.GetClusterVersion(),
		},
		{
			Tool:    mcp.MakeGetApiResourcesTool(),
			Handler: s.GetApiResources(),
		},
		{
			Tool:    mcp.MakeGetResourceDetailTool(),
			Handler: s.GetResourceDetailInfo(),
		},
		{
			Tool:    mcp.MakeListResourcesTool(),
			Handler: s.ListResources(),
		},
		{
			Tool:    mcp.MakeApplyResourceTool(),
			Handler: s.ApplyResource(),
		},
		{
			Tool:    mcp.MakeDeleteResourceTool(),
			Handler: s.DeleteResource(),
		},
		{
			Tool:    mcp.MakeGetPodLogsTool(),
			Handler: s.GetPodLogs(),
		},
		{
			Tool:    mcp.MakeRunInContainerTool(),
			Handler: s.RunInContainer(),
		},
		{
			Tool:    mcp.MakeTopPodTool(),
			Handler: s.TopPod(),
		},
		{
			Tool:    mcp.MakeTopNodeTool(),
			Handler: s.TopNode(),
		},
	}...)
}

// Start starts the mcp server.
func (s *Server) Start(ctx context.Context) error {
	s.RegisterTools(ctx)
	switch s.transport {
	case "sse":
		slog.Info("Starting mcp server with sse mode and listening on", "port", s.port)
		sseServer := server.NewSSEServer(s.svr, server.WithBaseURL(fmt.Sprintf("http://0.0.0.0:%d", s.port)))
		return sseServer.Start(fmt.Sprintf(":%d", s.port))
	case "stdio":
		slog.Info("Starting mcp server with STDIO mode")
		stdioServer := server.NewStdioServer(s.svr)
		return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
	}
	return errors.New("unsupported transport")
}
