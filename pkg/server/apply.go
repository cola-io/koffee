package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

// ApplyResource returns a function that applies a resource.
func (s *Server) ApplyResource() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		manifest, err := req.RequireString("manifest")
		if err != nil {
			return nil, err
		}

		namespace, enforceNamespace, err := s.cb.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return nil, err
		}

		builder := resource.NewBuilder(genericclioptions.NewConfigFlags(false))
		r := builder.Unstructured().
			ContinueOnError().
			NamespaceParam(namespace).DefaultNamespace().
			FilenameParam(enforceNamespace, nil). // TODO: Implement resource application logic
			Flatten().
			Do()

		if _, err := r.Infos(); err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(manifest), nil
	}
}
