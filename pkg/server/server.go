package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"cola.io/koffee/pkg/client"
	"cola.io/koffee/pkg/definition"
	"cola.io/koffee/pkg/tool"
	"cola.io/koffee/pkg/version"
)

type Option func(*Server)

type Server struct {
	svr       *mcp.Server
	generator *definition.HumanReadableGenerator
	cb        client.ClientBuilder
	transport string
	addr      string
}

// WithTransport sets the transport type for the server.
func WithTransport(t string) func(*Server) {
	return func(s *Server) {
		s.transport = t
	}
}

// WithAddr sets the address for the server when the transport is sse.
func WithAddr(addr string) func(*Server) {
	return func(s *Server) {
		s.addr = addr
	}
}

// NewServer creates a new mcp server.
func NewServer(kubeconfig string, opts ...Option) *Server {
	generator := definition.NewTableGenerator()
	definition.AddHandlers(generator)
	s := &Server{
		transport: "stdio",
		addr:      ":8888",
		svr: mcp.NewServer(&mcp.Implementation{
			Name:    version.Get().Module,
			Title:   "Kubernetes MCP Server",
			Version: version.Get().Version,
		}, nil),
		generator: generator,
		cb:        client.NewClientBuilder(kubeconfig),
	}

	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RegisterTools registers the tools for the server.
func (s *Server) RegisterTools() {
	slog.Info("Registering tools")
	mcp.AddTool(s.svr, tool.MakeListClusters(), s.ListClusters)
	mcp.AddTool(s.svr, tool.MakeSwitchContext(), s.SwitchContexts)
	mcp.AddTool(s.svr, tool.MakeGetClusterVersion(), s.GetClusterVersion)
	mcp.AddTool(s.svr, tool.MakeGetApiResources(), s.GetApiResources)
	mcp.AddTool(s.svr, tool.MakeGetResourceDetail(), s.GetResourceDetailInfo)
	mcp.AddTool(s.svr, tool.MakeListResources(), s.ListResources)
	mcp.AddTool(s.svr, tool.MakeDeleteResource(), s.DeleteResource)
	mcp.AddTool(s.svr, tool.MakeGetPodLogs(), s.GetPodLogs)
	mcp.AddTool(s.svr, tool.MakeRunInContainer(), s.RunInContainer)
	mcp.AddTool(s.svr, tool.MakeTopPod(), s.TopPod)
	mcp.AddTool(s.svr, tool.MakeTopNode(), s.TopNode)
}

// Start starts the mcp server.
func (s *Server) Start(ctx context.Context) error {
	s.RegisterTools()
	switch s.transport {
	case "sse":
		slog.Info("Starting mcp server with sse mode and listening on", "addr", s.addr)
		return http.ListenAndServe(s.addr, mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return s.svr
		}, nil))
	case "stdio":
		slog.Info("Starting mcp server with STDIO mode")
		return s.svr.Run(ctx, &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr})
	}
	return errors.New("unsupported transport")
}
