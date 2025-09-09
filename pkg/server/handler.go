package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"cola.io/koffee/pkg/definition"
)

// GetApiResourcesArgs represents the arguments for the GetApiResources tool.
type GetApiResourcesArgs struct {
	IncludeNamespaceScoped bool `json:"includeNamespaceScoped"`
}

// GetApiResourcesResult represents the result of the GetApiResources tool.
type GetApiResourcesResult []metav1.APIResource

func (s *Server) GetApiResources(ctx context.Context, req *mcp.CallToolRequest, args *GetApiResourcesArgs) (*mcp.CallToolResult, GetApiResourcesResult, error) {
	includeNamespaceScoped := args.IncludeNamespaceScoped

	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, nil, err
	}

	resources, err := ListApiResources(discoveryClient, includeNamespaceScoped)
	if err != nil {
		return nil, nil, err
	}

	out, err := json.Marshal(resources)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(out)},
		},
		StructuredContent: resources,
	}, resources, nil
}

// GetResourceDetailInfoArgs represents the arguments for the GetResourceDetailInfo tool.
type GetResourceDetailInfoArgs struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (s *Server) GetResourceDetailInfo(ctx context.Context, req *mcp.CallToolRequest, args *GetResourceDetailInfoArgs) (*mcp.CallToolResult, *unstructured.Unstructured, error) {
	kind := args.Kind
	resourceName := args.Name
	namespace := args.Namespace

	slog.Info("Getting resource detail info", "kind", kind, "name", resourceName, "namespace", namespace)

	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, nil, err
	}

	dynamicClient, err := s.cb.GetDynamicClient()
	if err != nil {
		return nil, nil, err
	}

	gvResource, err := lookupGroupVersionResource(discoveryClient, kind)
	if err != nil {
		return nil, nil, err
	}

	var obj *unstructured.Unstructured
	if len(namespace) > 0 {
		obj, err = dynamicClient.Resource(gvResource).Namespace(namespace).Get(ctx, resourceName, metav1.GetOptions{})
	} else {
		obj, err = dynamicClient.Resource(gvResource).Get(ctx, resourceName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get resource info: %w", err)
	}
	obj.SetManagedFields(nil)

	out, err := obj.MarshalJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal resource: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(out)},
		},
		StructuredContent: obj,
	}, obj, nil
}

// ListResourcesArgs represents the arguments for listing resources.
type ListResourcesArgs struct {
	Kind          string `json:"kind"`
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector"`
	FieldSelector string `json:"fieldSelector"`
}

func (s *Server) ListResources(ctx context.Context, req *mcp.CallToolRequest, args *ListResourcesArgs) (*mcp.CallToolResult, *metav1.Table, error) {
	kind := args.Kind
	namespace := args.Namespace
	labelSelector := args.LabelSelector
	fieldSelector := args.FieldSelector

	slog.Info("Listing resources", "kind", kind, "namespace", namespace, "labelSelector", labelSelector, "fieldSelector", fieldSelector)

	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, nil, err
	}

	if _, err = lookupGroupVersionResource(discoveryClient, kind); err != nil {
		return nil, nil, err
	}

	gvResource, err := lookupGroupVersionResource(discoveryClient, kind)
	if err != nil {
		return nil, nil, err
	}

	dynamicClient, err := s.cb.GetDynamicClient()
	if err != nil {
		return nil, nil, err
	}

	var options metav1.ListOptions
	if len(labelSelector) > 0 {
		options.LabelSelector = labelSelector
	}
	if len(fieldSelector) > 0 {
		options.FieldSelector = fieldSelector
	}

	var items *unstructured.UnstructuredList
	if len(namespace) > 0 {
		items, err = dynamicClient.Resource(gvResource).Namespace(namespace).List(ctx, options)
	} else {
		items, err = dynamicClient.Resource(gvResource).List(ctx, options)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list resources: %w", err)
	}

	slog.Info("Listing resources", "kind", kind, "namespace", namespace, "items", len(items.Items))

	obj, supported := definition.IsSupportedKind(kind)
	table := &metav1.Table{}
	if supported {
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(items.UnstructuredContent(), obj); err != nil {
			return nil, nil, err
		}

		table, err = s.generator.GenerateTable(obj)
		if err != nil {
			return nil, nil, err
		}
	} else {
		table.ColumnDefinitions = []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Namespace", Type: "string"},
			{Name: "Age", Type: "string"},
		}

		rows := make([]metav1.TableRow, 0)
		for _, item := range items.Items {
			row := metav1.TableRow{
				Cells: make([]any, 0),
			}
			row.Cells = append(row.Cells, item.GetName(), item.GetNamespace(), time.Since(item.GetCreationTimestamp().Time))
			rows = append(rows, row)
		}
		table.Rows = rows
	}

	out, err := json.Marshal(table)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(out)},
		},
		StructuredContent: table,
	}, table, nil
}

func ListApiResources(discoveryClient discovery.DiscoveryInterface, includeNamespaceScoped bool) ([]metav1.APIResource, error) {
	// list all api resources in cluster
	apiResources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to list api resources: %w", err)
	}

	var resources []metav1.APIResource
	for _, apiResource := range apiResources {
		groupVersion := apiResource.GroupVersion
		for _, resource := range apiResource.APIResources {
			if len(resource.Group) == 0 {
				resource.Group = apiResource.GroupVersion
			}

			if len(resource.Version) == 0 {
				gv, err := schema.ParseGroupVersion(groupVersion)
				if err != nil {
					continue
				}
				resource.Version = gv.Version
			}

			// filter the non-namespaced resource
			if resource.Namespaced && !includeNamespaceScoped {
				continue
			}
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func lookupGroupVersionResource(discoveryClient discovery.DiscoveryInterface, kind string) (schema.GroupVersionResource, error) {
	apiResources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	for _, apiResource := range apiResources {
		gv, err := schema.ParseGroupVersion(apiResource.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range apiResource.APIResources {
			if resource.Kind != kind {
				continue
			}
			return schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: resource.Name,
			}, nil
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("not found resource for kind %q", kind)
}
