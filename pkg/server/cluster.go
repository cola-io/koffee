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

type ClusterContexts []ClusterContext

func (s *Server) ListClusters(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[any]) (*mcp.CallToolResultFor[ClusterContexts], error) {
	cfg, err := s.cb.ToRawKubeConfigLoader().RawConfig()
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

	result, err := json.Marshal(ctxs)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[ClusterContexts]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(result)},
		},
		StructuredContent: ctxs,
	}, nil
}

// SwitchContextArgs represents the arguments for switching cluster contexts.
type SwitchContextArgs struct {
	Name string `json:"name" mcp:"The name of the cluster context to switch to"`
}

func (s *Server) SwitchContexts(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[SwitchContextArgs]) (*mcp.CallToolResultFor[any], error) {
	cfg, err := s.cb.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return nil, err
	}

	inputContext := req.Arguments.Name

	slog.Info("Loading contexts", "inputContext", inputContext)

	if _, ok := cfg.Contexts[inputContext]; !ok {
		return nil, fmt.Errorf("context %q not found in the specified kuebconfig", inputContext)
	}

	cfg.CurrentContext = inputContext
	if err = s.cb.WriteToFile(cfg); err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "switch cluster context successfully"},
		},
	}, nil
}

func (s *Server) GetClusterVersion(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[any]) (*mcp.CallToolResultFor[*version.Info], error) {
	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, err
	}

	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(serverVersion)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[*version.Info]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(out)},
		},
		StructuredContent: serverVersion,
	}, nil
}
