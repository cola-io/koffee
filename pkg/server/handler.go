package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"cola.io/koffee/pkg/definition"
)

func (s *Server) GetResourceDetailInfo() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		resp, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(resp)), nil
	}
}

func (s *Server) GetApiResources() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		includeNamespaceScoped := req.GetBool("includeNamespaceScoped", true)

		discoveryClient, err := s.cb.GetDiscoveryClient()
		if err != nil {
			return nil, err
		}

		resources, err := ListApiResources(discoveryClient, includeNamespaceScoped)
		if err != nil {
			return nil, err
		}

		resp, err := json.Marshal(resources)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(resp)), nil
	}
}

func (s *Server) ListResources() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		kind, err := req.RequireString("kind")
		if err != nil {
			return nil, err
		}
		namespace := req.GetString("namespace", "")
		labelSelector := req.GetString("labelSelector", "")
		fieldSelector := req.GetString("fieldSelector", "")

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

		out, err := json.Marshal(table)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	}
}

func ListApiResources(discoveryClient discovery.DiscoveryInterface, includeNamespaceScoped bool) ([]map[string]any, error) {
	// list all api resources in cluster
	apiResources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to list api resources: %w", err)
	}

	var resources []map[string]any
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

			resources = append(resources, map[string]any{
				"name":         resource.Name,
				"singularName": resource.SingularName,
				"namespaced":   resource.Namespaced,
				"kind":         resource.Kind,
				"group":        resource.Group,
				"version":      resource.Version,
				"verbs":        resource.Verbs,
			})
		}
	}
	return resources, nil
}

func (s *Server) CreateResource() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		kind, err := req.RequireString("kind")
		if err != nil {
			return nil, err
		}

		manifest, err := req.RequireString("manifest")
		if err != nil {
			return nil, err
		}
		namespace := req.GetString("namespace", "")

		slog.Info("Loading create resource", "kind", kind, "namespace", namespace, "manifest", manifest)

		discoveryClient, err := s.cb.GetDiscoveryClient()
		if err != nil {
			return nil, err
		}

		gvr, err := lookupGroupVersionResource(discoveryClient, kind)
		if err != nil {
			return nil, err
		}

		obj := &unstructured.Unstructured{}
		if err = json.Unmarshal([]byte(manifest), &obj.Object); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
		}

		dynamicClient, err := s.cb.GetDynamicClient()
		if err != nil {
			return nil, err
		}

		var result *unstructured.Unstructured
		if len(namespace) > 0 || len(obj.GetNamespace()) > 0 {
			targetNamespace := namespace
			if targetNamespace == "" {
				targetNamespace = obj.GetNamespace()
			}
			result, err = dynamicClient.Resource(gvr).Namespace(targetNamespace).Create(ctx, obj, metav1.CreateOptions{})
		} else {
			result, err = dynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create resource: %w", err)
		}

		resp, err := json.Marshal(result.UnstructuredContent())
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(resp)), nil
	}
}

func (s *Server) UpdateResource() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		kind, err := req.RequireString("kind")
		if err != nil {
			return nil, err
		}

		resourceName, err := req.RequireString("name")
		if err != nil {
			return nil, err
		}

		manifest, err := req.RequireString("manifest")
		if err != nil {
			return nil, err
		}
		namespace := req.GetString("namespace", "")

		slog.Info("Loading update resource", "kind", kind, "namespace", namespace, "name", resourceName, "manifest", manifest)

		obj := &unstructured.Unstructured{}
		if err = json.Unmarshal([]byte(manifest), &obj.Object); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
		}

		if obj.GetName() != resourceName {
			return nil, fmt.Errorf("failed to update resource due to the name is mismatch the object")
		}

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

		var result *unstructured.Unstructured
		if len(namespace) > 0 {
			result, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		} else {
			result, err = dynamicClient.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
		}
		if err != nil {
			return nil, fmt.Errorf("failed to update resource: %w", err)
		}

		resp, err := json.Marshal(result.UnstructuredContent())
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(resp)), nil
	}
}

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
