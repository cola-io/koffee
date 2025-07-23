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
	Kind      string `json:"kind" mcp:"The type of the specified resource"`
	Name      string `json:"name" mcp:"The name of the specified resource"`
	Namespace string `json:"namespace" mcp:"The namespace of the resource, (required for namespace-scoped resources)"`
}

func (s *Server) DeleteResource(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[DeleteResourceArgs]) (*mcp.CallToolResultFor[any], error) {
	kind := req.Arguments.Kind
	resourceName := req.Arguments.Name
	namespace := req.Arguments.Namespace

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

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "delete resource successfully"},
		},
	}, nil
}
