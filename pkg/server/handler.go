package server

import (
	"context"
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
	IncludeNamespaceScoped bool `json:"includeNamespaceScoped" mcp:"Include namespace-scoped resources"`
}

// GetApiResourcesResult represents the result of the GetApiResources tool.
type GetApiResourcesResult struct {
	Resources []metav1.APIResource `json:"resources"`
}

func (s *Server) GetApiResources(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[GetApiResourcesArgs]) (*mcp.CallToolResultFor[GetApiResourcesResult], error) {
	includeNamespaceScoped := req.Arguments.IncludeNamespaceScoped

	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, err
	}

	resources, err := ListApiResources(discoveryClient, includeNamespaceScoped)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[GetApiResourcesResult]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "get api resources successfully"},
		},
		StructuredContent: GetApiResourcesResult{resources},
	}, nil
}

type GetResourceDetailInfoArgs struct {
	Kind      string `json:"kind" mcp:"Resource type"`
	Name      string `json:"name" mcp:"The name of the resource to get information about"`
	Namespace string `json:"namespace" mcp:"Namespace (required for namespace-scoped resources)"`
}

func (s *Server) GetResourceDetailInfo(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[GetResourceDetailInfoArgs]) (*mcp.CallToolResultFor[*unstructured.Unstructured], error) {
	kind := req.Arguments.Kind
	resourceName := req.Arguments.Name
	namespace := req.Arguments.Namespace

	slog.Info("Getting resource detail info", "kind", kind, "name", resourceName, "namespace", namespace)

	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := s.cb.GetDynamicClient()
	if err != nil {
		return nil, err
	}

	gvResource, err := lookupGroupVersionResource(discoveryClient, kind)
	if err != nil {
		return nil, err
	}

	var obj *unstructured.Unstructured
	if len(namespace) > 0 {
		obj, err = dynamicClient.Resource(gvResource).Namespace(namespace).Get(ctx, resourceName, metav1.GetOptions{})
	} else {
		obj, err = dynamicClient.Resource(gvResource).Get(ctx, resourceName, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get resource info: %w", err)
	}
	obj.SetManagedFields(nil)

	return &mcp.CallToolResultFor[*unstructured.Unstructured]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "get api resources successfully"},
		},
		StructuredContent: obj,
	}, nil
}

type ListResourcesArgs struct {
	Kind          string `json:"kind" mcp:"Resource type"`
	Namespace     string `json:"namespace" mcp:"The namespace of the resource, If non-empty, only list resources in this namespace"`
	LabelSelector string `json:"labelSelector" mcp:"LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints"`
	FieldSelector string `json:"fieldSelector" mcp:"FieldSelector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type"`
}

func (s *Server) ListResources(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[ListResourcesArgs]) (*mcp.CallToolResultFor[*metav1.Table], error) {
	kind := req.Arguments.Kind
	namespace := req.Arguments.Namespace
	labelSelector := req.Arguments.LabelSelector
	fieldSelector := req.Arguments.FieldSelector

	slog.Info("Listing resources", "kind", kind, "namespace", namespace, "labelSelector", labelSelector, "fieldSelector", fieldSelector)

	discoveryClient, err := s.cb.GetDiscoveryClient()
	if err != nil {
		return nil, err
	}

	if _, err = lookupGroupVersionResource(discoveryClient, kind); err != nil {
		return nil, err
	}

	gvResource, err := lookupGroupVersionResource(discoveryClient, kind)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := s.cb.GetDynamicClient()
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	slog.Info("Listing resources", "kind", kind, "namespace", namespace, "items", len(items.Items))

	obj, supported := definition.IsSupportedKind(kind)
	table := &metav1.Table{}
	if supported {
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(items.UnstructuredContent(), obj); err != nil {
			return nil, err
		}

		table, err = s.generator.GenerateTable(obj)
		if err != nil {
			return nil, err
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

	return &mcp.CallToolResultFor[*metav1.Table]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "list resources successfully"},
		},
		StructuredContent: table,
	}, nil
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
