package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Server) DeleteResource() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		kind, err := req.RequireString("kind")
		if err != nil {
			return nil, err
		}

		resourceName, err := req.RequireString("name")
		if err != nil {
			return nil, err
		}
		namespace := req.GetString("namespace", "")

		slog.Info("Loading delete resource", "kind", kind, "name", resourceName, "namespace", namespace)

		discoveryClient, err := s.cb.GetDiscoveryClient()
		if err != nil {
			return nil, err
		}

		gvr, err := lookupGroupVersionResource(discoveryClient, kind)
		if err != nil {
			return nil, err
		}

		dynamicClient, err := s.cb.GetDynamicClient()
		if err != nil {
			return nil, err
		}

		if len(namespace) > 0 {
			err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, resourceName, metav1.DeleteOptions{})
		} else {
			err = dynamicClient.Resource(gvr).Delete(ctx, resourceName, metav1.DeleteOptions{})
		}
		if err != nil {
			return nil, fmt.Errorf("failed to delete resource: %w", err)
		}
		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted resource %s/%s", kind, resourceName)), nil
	}
}
