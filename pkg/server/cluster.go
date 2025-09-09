package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/apimachinery/pkg/version"
)

type ClusterContext struct {
	Name        string `json:"name,omitempty"`
	Current     bool   `json:"current,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	User        string `json:"user,omitempty"`
	Server      string `json:"server,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

type ClusterContexts struct {
	Contexts []ClusterContext `json:"contexts,omitempty"`
}

func (s *Server) ListClusters(ctx context.Context, req *mcp.CallToolRequest, param any) (*mcp.CallToolResult, *ClusterContexts, error) {
	cfg, err := s.cb.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return nil, nil, err
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

	result, err := json.Marshal(ctxs)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(result)},
		},
		StructuredContent: ctxs,
	}, &ClusterContexts{Contexts: ctxs}, nil
}

// SwitchContextArgs represents the arguments for switching cluster contexts.
type SwitchContextArgs struct {
	Name string `json:"name"`
}

func (s *Server) SwitchContexts(ctx context.Context, req *mcp.CallToolRequest, args *SwitchContextArgs) (*mcp.CallToolResult, any, error) {
	cfg, err := s.cb.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return nil, nil, err
	}

	inputContext := args.Name

	slog.Info("Loading contexts", "inputContext", inputContext)

	if _, ok := cfg.Contexts[inputContext]; !ok {
		return nil, nil, fmt.Errorf("context %q not found in the specified kuebconfig", inputContext)
	}

	cfg.CurrentContext = inputContext
	if err = s.cb.WriteToFile(cfg); err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "switch cluster context successfully"},
		},
	}, nil, nil
}

func (s *Server) GetClusterVersion(ctx context.Context, req *mcp.CallToolRequest, args any) (*mcp.CallToolResult, *version.Info, error) {
	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, nil, err
	}

	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, nil, err
	}

	out, err := json.Marshal(serverVersion)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(out)},
		},
		StructuredContent: serverVersion,
	}, serverVersion, nil
}
