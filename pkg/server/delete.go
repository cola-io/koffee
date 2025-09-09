package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteResourceArgs represents the arguments for deleting a resource.
type DeleteResourceArgs struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (s *Server) DeleteResource(ctx context.Context, req *mcp.CallToolRequest, args *DeleteResourceArgs) (*mcp.CallToolResult, any, error) {
	kind := args.Kind
	resourceName := args.Name
	namespace := args.Namespace

	slog.Info("Loading delete resource", "kind", kind, "name", resourceName, "namespace", namespace)

	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, nil, err
	}

	gvr, err := lookupGroupVersionResource(discoveryClient, kind)
	if err != nil {
		return nil, nil, err
	}

	dynamicClient, err := s.cb.GetDynamicClient()
	if err != nil {
		return nil, nil, err
	}

	if len(namespace) > 0 {
		err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, resourceName, metav1.DeleteOptions{})
	} else {
		err = dynamicClient.Resource(gvr).Delete(ctx, resourceName, metav1.DeleteOptions{})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete resource: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "delete resource successfully"},
		},
	}, nil, nil
}
