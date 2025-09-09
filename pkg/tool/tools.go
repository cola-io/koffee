package tool

import (
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"
)

// MakeListClusters creates a tool for listing the all Kubernetes clusters
func MakeListClusters() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_clusters",
		Title:       "List Clusters",
		Description: "List the local kube cluster context information",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
		OutputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"contexts": {
					Type:        "array",
					Description: "",
					Items: &jsonschema.Schema{
						Properties: map[string]*jsonschema.Schema{
							"name": {
								Type:        "string",
								Description: "The name of the cluster context",
							},
							"current": {
								Type:        "string",
								Description: "The current cluster",
							},
							"cluster_name": {
								Type:        "string",
								Description: "The cluster name",
							},
							"user": {
								Type:        "string",
								Description: "The current user",
							},
							"server": {
								Type:        "string",
								Description: "The server address",
							},
							"namespace": {
								Type:        "string",
								Description: "The namespace",
							},
						},
						Required: []string{"name", "current", "cluster_name", "server"},
					},
				},
			},
			Required: []string{"contexts"},
		},
	}
}

// MakeSwitchContext creates a tool for switching the Kubernetes context
func MakeSwitchContext() *mcp.Tool {
	return &mcp.Tool{
		Name:        "switch_context",
		Title:       "Switch Context",
		Description: "Switch the kubernetes context",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   false,
			IdempotentHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The name of the cluster context to switch to",
				},
			},
			Required: []string{"name"},
		},
	}
}

// MakeGetClusterVersion creates a tool for getting the cluster version
func MakeGetClusterVersion() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_cluster_version",
		Title:       "Get Cluster Version",
		Description: "Get the cluster version",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
	}
}

// MakeGetApiResources creates a tool for getting API resources
func MakeGetApiResources() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_api_resources",
		Title:       "Get API Resources",
		Description: "Get all supported API resource types in the cluster, including built-in resources and CRDs",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"includeNamespaceScoped": {
					Type:        "boolean",
					Description: "Whether to include namespace-scoped resources",
				},
			},
		},
		OutputSchema: &jsonschema.Schema{
			Type: "object",
		},
	}
}

// MakeGetResourceDetail creates a tool for getting a specific resource
func MakeGetResourceDetail() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_resource_detail",
		Title:       "Get Resource Detail Information",
		Description: "Get detailed information about a specific resource",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"kind": {
					Type:        "string",
					Description: "The kind of resource",
				},
				"name": {
					Type:        "string",
					Description: "The name of the resource",
				},
				"namespace": {
					Type:        "string",
					Description: "Namespace (required for namespace-scoped resources)",
				},
			},
			Required: []string{"kind", "name"},
		},
	}
}

// MakeListResources creates a tool for listing resources
func MakeListResources() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_resources",
		Title:       "List Resources",
		Description: "List all instances of a resource type",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"kind": {
					Type:        "string",
					Description: "The kind of resource",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the resource, If non-empty, only list resources in this namespace",
				},
				"labelSelector": {
					Type: "string",
					Description: `LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching
					                objects must satisfy all of the specified label constraints`,
				},
				"fieldSelector": {
					Type: "string",
					Description: `FieldSelector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector
									key1=value1,key2=value2). The server only supports a limited number of field queries per type`,
				},
			},
			Required: []string{"kind"},
		},
	}
}

// MakeApplyResourceTool creates a tool for applying resources, like `kubectl apply -f <manifest>`
func MakeApplyResourceTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "apply_resource",
		Title:       "Apply Resource",
		Description: "Apply a configuration to a resource by file name. The resource name must be specified. This resource will be created if it doesn't exist yet",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"manifest": {
					Type:        "string",
					Description: "Resource manifest, JSON and YAML formats are accepted",
				},
			},
			Required: []string{"manifest"},
		},
	}
}

// MakeDeleteResource creates a tool for deleting resources
func MakeDeleteResource() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_resource",
		Title:       "Delete Resource",
		Description: "Delete a resource with the specified name and namespace if it's namespace-scoped",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"kind": {
					Type:        "string",
					Description: "The type of the specified resource",
				},
				"name": {
					Type:        "string",
					Description: "The name of the specified resource",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the resource, (required for namespace-scoped resources)",
				},
			},
			Required: []string{"kind", "name"},
		},
	}
}

// MakeGetPodLogs creates a tool for getting pod logs
func MakeGetPodLogs() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_pod_logs",
		Description: "Get the logs for a container in a pod or specified resource. If the pod has only one container, the container name is optional",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The specified pod name",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the pod",
				},
				"container": {
					Type:        "string",
					Description: "Get the logs of this container in the pod",
				},
				"tail": {
					Type:        "integer",
					Description: "Lines of recent log file to display",
					Minimum:     ptr.To(1.0),
					Maximum:     ptr.To(100.0),
				},
			},
			Required: []string{"kind", "name", "namespace"},
		},
	}
}

// MakeRunInContainer creates a tool for executing commands in a pod
func MakeRunInContainer() *mcp.Tool {
	return &mcp.Tool{
		Name:        "run_in_container",
		Title:       "Run In Container",
		Description: "Execute a command in a container. If the container is empty, it uses the default container or the first container in the pod.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: false,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "Name of the Pod where the command will be executed",
				},
				"namespace": {
					Type:        "string",
					Description: "Namespace of the Pod where the command will be executed",
				},
				"container": {
					Type:        "string",
					Description: "The container name which execute command in the pod",
				},
				"command": {
					Type:        "array",
					Items:       &jsonschema.Schema{Type: "string"},
					Description: "Command to execute in the Pod container",
				},
			},
			Required: []string{"name", "namespace", "container", "command"},
		},
	}
}

// MakeTopPod creates a tool for displaying resource (CPU/memory) usage of pods.
func MakeTopPod() *mcp.Tool {
	return &mcp.Tool{
		Name:        "top_pod",
		Title:       "Top Pod",
		Description: "Display resource (CPU/memory) usage of pods. It allows you to see the resource consumption of pods",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The specified pod name",
				},
				"namespace": {
					Type:        "string",
					Description: "The namespace of the pod",
				},
				"sortBy": {
					Type:        "string",
					Description: "If non-empty, sort pods list using specified field. The field can be either 'cpu' or 'memory'",
				},
				"labelSelector": {
					Type: "string",
					Description: `LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching
									objects must satisfy all of the specified label constraints`,
				},
				"fieldSelector": {
					Type: "string",
					Description: `FieldSelector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector
									key1=value1,key2=value2). The server only supports a limited number of field queries per type`,
				},
			},
		},
	}
}

// MakeTopNode creates a tool for displaying resource (CPU/memory) usage of nodes.
func MakeTopNode() *mcp.Tool {
	return &mcp.Tool{
		Name:        "top_node",
		Title:       "Top Node",
		Description: "Display resource (CPU/memory) usage of nodes. It allows you to see the resource consumption of nodes",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "The specified node name",
				},
				"sortBy": {
					Type:        "string",
					Description: "If non-empty, sort pods list using specified field. The field can be either 'cpu' or 'memory'",
				},
				"labelSelector": {
					Type: "string",
					Description: `LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching
									objects must satisfy all of the specified label constraints`,
				},
			},
		},
	}
}
