package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
)

type ClusterContext struct {
	Name        string `json:"name,omitempty"`
	Current     bool   `json:"current,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	User        string `json:"user,omitempty"`
	Server      string `json:"server,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

func (s *Server) ListClusters() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, err := s.cb.LoadRawConfig()
		if err != nil {
			return nil, err
		}

		ctxs := make([]ClusterContext, 0)
		for name, ctx := range cfg.Contexts {
			if ctx.Namespace == "" {
				ctx.Namespace = "default"
			}
			current := name == cfg.CurrentContext
			ctxs = append(ctxs, ClusterContext{
				Name:        name,
				Current:     current,
				ClusterName: ctx.Cluster,
				User:        ctx.AuthInfo,
				Server:      cfg.Clusters[ctx.Cluster].Server,
				Namespace:   ctx.Namespace,
			})
		}

		resp, err := json.Marshal(ctxs)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(resp)), nil
	}
}

func (s *Server) SwitchContexts() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, err := s.cb.LoadRawConfig()
		if err != nil {
			return nil, err
		}

		inputContext, err := req.RequireString("name")
		if err != nil {
			return nil, err
		}

		slog.Info("Loading contexts", "inputContext", inputContext)

		if _, ok := cfg.Contexts[inputContext]; !ok {
			return nil, fmt.Errorf("context %q not found in the specified kuebconfig", inputContext)
		}

		cfg.CurrentContext = inputContext
		if err = s.cb.WriteToFile(*cfg); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText("switch cluster context successful"), nil
	}
}

func (s *Server) GetClusterVersion() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		discoveryClient, err := s.cb.GetDiscoveryClient()
		if err != nil {
			return nil, err
		}

		serverVersion, err := discoveryClient.ServerVersion()
		if err != nil {
			return nil, err
		}

		resp, err := json.Marshal(serverVersion)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(resp)), nil
	}
}
