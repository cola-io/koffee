package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

// ApplyResourceArgs represents the arguments for applying a resource.
type ApplyResourceArgs struct {
	Manifest string `json:"manifest" mcp:"Resource manifest, JSON and YAML formats are accepted"`
}

// ApplyResource returns a function that applies a resource.
func (s *Server) ApplyResource(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[ApplyResourceArgs]) (*mcp.CallToolResultFor[any], error) {
	manifest := req.Arguments.Manifest

	namespace, enforceNamespace, err := s.cb.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}

	builder := resource.NewBuilder(genericclioptions.NewConfigFlags(false))
	r := builder.Unstructured().
		ContinueOnError().
		NamespaceParam(namespace).
		FilenameParam(enforceNamespace, nil). // TODO: Implement resource application logic
		Flatten().
		Do()

	if _, err := r.Infos(); err != nil {
		return nil, err
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "apply manifest successfully"},
		},
		StructuredContent: manifest,
	}, nil
}
