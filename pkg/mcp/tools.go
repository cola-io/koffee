package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// MakeListClustersTool creates a tool for listing the all Kubernetes clusters
func MakeListClustersTool() mcp.Tool {
	return mcp.NewTool("list_clusters",
		mcp.WithDescription("List the local kube cluster context information"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeSwitchContextTool creates a tool for switching the Kubernetes context
func MakeSwitchContextTool() mcp.Tool {
	return mcp.NewTool("switch_context",
		mcp.WithDescription("Switch the Kubernetes context"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the cluster context to switch to"),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeGetClusterVersionTool creates a tool for getting the cluster version
func MakeGetClusterVersionTool() mcp.Tool {
	return mcp.NewTool("get_cluster_version",
		mcp.WithDescription("Get the cluster version"),
		mcp.WithString("name",
			mcp.Description("The name of the context to get cluster version"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
}

// MakeGetApiResourcesTool creates a tool for getting API resources
func MakeGetApiResourcesTool() mcp.Tool {
	return mcp.NewTool("get_api_resources",
		mcp.WithDescription("Get all supported API resource types in the cluster, including built-in resources and CRDs"),
		mcp.WithBoolean("includeNamespaceScoped",
			mcp.Description("Include namespace-scoped resources"),
			mcp.DefaultBool(true),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeGetResourceDetailTool creates a tool for getting a specific resource
func MakeGetResourceDetailTool() mcp.Tool {
	return mcp.NewTool("get_resource_detail",
		mcp.WithDescription("Get detailed information about a specific resource"),
		mcp.WithString("kind",
			mcp.Required(),
			mcp.Description("Resource type"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the resource to get information about"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace (required for namespace-scoped resources)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeListResourcesTool creates a tool for listing resources
func MakeListResourcesTool() mcp.Tool {
	return mcp.NewTool("list_resources",
		mcp.WithDescription("List all instances of a resource type"),
		mcp.WithString("kind",
			mcp.Required(),
			mcp.Description("Resource type"),
		),
		mcp.WithString("namespace",
			mcp.Description("The namespace of the resource, If non-empty, only list resources in this namespace"),
		),
		mcp.WithString("labelSelector",
			mcp.Description(`LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching
				objects must satisfy all of the specified label constraints`),
		),
		mcp.WithString("fieldSelector",
			mcp.Description(`FieldSelector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector
				key1=value1,key2=value2). The server only supports a limited number of field queries per type`),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeApplyResourceTool creates a tool for applying resources, like `kubectl apply -f <manifest>`
func MakeApplyResourceTool() mcp.Tool {
	return mcp.NewTool("apply_resource",
		mcp.WithDescription(`Apply a configuration to a resource by file name. The resource name must be specified. This resource will be
created if it doesn't exist yet`),
		mcp.WithString("manifest",
			mcp.Required(),
			mcp.Description("Resource manifest, JSON and YAML formats are accepted"),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeDeleteResourceTool creates a tool for deleting resources
func MakeDeleteResourceTool() mcp.Tool {
	return mcp.NewTool("delete_resource",
		mcp.WithDescription("Delete a resource with the specified name and namespace if it's namespace-scoped"),
		mcp.WithString("kind",
			mcp.Required(),
			mcp.Description("The type of the specified resource"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the specified resource"),
		),
		mcp.WithString("namespace",
			mcp.Description("The namespace of the resource, (required for namespace-scoped resources)"),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeGetPodLogsTool creates a tool for getting pod logs
func MakeGetPodLogsTool() mcp.Tool {
	return mcp.NewTool("get_pod_logs",
		mcp.WithDescription(`Get the logs for a container in a pod or specified resource. If the pod has only one container, the container name is
optional`),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The specified pod name"),
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("The namespace of the pod"),
		),
		mcp.WithString("container",
			mcp.Description("Get the logs of this container in the pod"),
		),
		mcp.WithNumber("tail",
			mcp.DefaultNumber(50),
			mcp.Min(1.0),
			mcp.Max(100.0),
			mcp.Description("Lines of recent log file to display"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeRunInContainerTool creates a tool for executing commands in a pod
func MakeRunInContainerTool() mcp.Tool {
	return mcp.NewTool("run_in_container",
		mcp.WithDescription(`Execute a command in a container.
		If the container is empty, it uses the default container or the first container in the pod.`),
		mcp.WithString("name",
			mcp.Description("Name of the Pod where the command will be executed"),
			mcp.Required(),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace of the Pod where the command will be executed"),
		),
		mcp.WithString("container",
			mcp.Description("The container name which execute command in the pod"),
		),
		mcp.WithArray("command",
			mcp.Description("Command to execute in the Pod container."),
			mcp.Required(),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeTopPodTool creates a tool for displaying resource (CPU/memory) usage of pods.
func MakeTopPodTool() mcp.Tool {
	return mcp.NewTool("top_pod",
		mcp.WithDescription(`Display resource (CPU/memory) usage of pods. It allows you to see the resource consumption of pods`),
		mcp.WithString("name",
			mcp.Description("The specified pod name"),
		),
		mcp.WithString("namespace",
			mcp.Description("The namespace of the pod"),
		),
		mcp.WithString("sortBy",
			mcp.Description("If non-empty, sort pods list using specified field. The field can be either 'cpu' or 'memory'."),
		),
		mcp.WithString("labelSelector",
			mcp.Description(`LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching
				objects must satisfy all of the specified label constraints`),
		),
		mcp.WithString("fieldSelector",
			mcp.Description(`FieldSelector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector
				key1=value1,key2=value2). The server only supports a limited number of field queries per type`),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// MakeTopNodeTool creates a tool for displaying resource (CPU/memory) usage of nodes.
func MakeTopNodeTool() mcp.Tool {
	return mcp.NewTool("top_node",
		mcp.WithDescription(`Display resource (CPU/memory) usage of nodes. It allows you to see the resource consumption of nodes`),
		mcp.WithString("name",
			mcp.Description("The specified node name"),
		),
		mcp.WithString("sortBy",
			mcp.Description("If non-empty, sort nodes list using specified field. The field can be either 'cpu' or 'memory'."),
		),
		mcp.WithString("labelSelector",
			mcp.Description(`LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching
				objects must satisfy all of the specified label constraints`),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
