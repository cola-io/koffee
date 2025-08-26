package server

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/metricsutil"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// TopPodArgs represents the arguments for the top pod tool.
type TopPodArgs struct {
	Namespace     string `json:"namespace" jsonschema:"The namespace of the pod"`
	Name          string `json:"name" jsonschema:"The specified pod name"`
	SortBy        string `json:"sortBy" jsonschema:"If non-empty, sort pods list using specified field. The field can be either 'cpu' or 'memory'"`
	LabelSelector string `json:"labelSelector" jsonschema:"LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints"`
	FieldSelector string `json:"fieldSelector" jsonschema:"FieldSelector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type"`
}

func (s *Server) TopPod(ctx context.Context, req *mcp.CallToolRequest, args *TopPodArgs) (*mcp.CallToolResult, *bytes.Buffer, error) {
	namespace := args.Namespace
	resourceName := args.Name
	sortBy := args.SortBy
	labelSelector := args.LabelSelector
	fieldSelector := args.FieldSelector

	slog.Info("Loading top pod argument", "namespace", namespace, "resourceName", resourceName, "sortBy", sortBy, "labelSelector", labelSelector, "fieldSelector", fieldSelector)

	metricClient, err := s.cb.GetMetricsClient()
	if err != nil {
		return nil, nil, err
	}

	versionedMetrics := &metricsv1beta1.PodMetricsList{}
	if resourceName != "" {
		m, err := metricClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}
		versionedMetrics.Items = []metricsv1beta1.PodMetrics{*m}
	} else {
		options := metav1.ListOptions{}
		if len(labelSelector) > 0 {
			options.LabelSelector = labelSelector
		}
		if len(fieldSelector) > 0 {
			options.FieldSelector = fieldSelector
		}
		versionedMetrics, err = metricClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, options)
		if err != nil {
			return nil, nil, err
		}
	}

	metrics := &metricsapi.PodMetricsList{}
	if err = metricsv1beta1.Convert_v1beta1_PodMetricsList_To_metrics_PodMetricsList(versionedMetrics, metrics, nil); err != nil {
		return nil, nil, err
	}

	out := bytes.NewBuffer(make([]byte, 0))
	if err := metricsutil.NewTopCmdPrinter(out).PrintPodMetrics(metrics.Items, true, true, false, sortBy, true); err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: out.String()},
		},
		StructuredContent: out,
	}, out, nil
}

// TopNodeArgs represents the arguments for the TopNode command.
type TopNodeArgs struct {
	Name          string `json:"name" jsonschema:"The specified node name"`
	SortBy        string `json:"sortBy" jsonschema:"If non-empty, sort nodes list using specified field. The field can be either 'cpu' or 'memory'"`
	LabelSelector string `json:"labelSelector" jsonschema:"LabelSelector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints"`
}

func (s *Server) TopNode(ctx context.Context, req *mcp.CallToolRequest, args *TopNodeArgs) (*mcp.CallToolResult, *bytes.Buffer, error) {
	resourceName := args.Name
	sortBy := args.SortBy
	labelSelector := args.LabelSelector

	slog.Info("Loading top node argument", "resourceName", resourceName, "sortBy", sortBy, "labelSelector", labelSelector)

	cli, err := s.cb.GetClient()
	if err != nil {
		return nil, nil, err
	}

	metricClient, err := s.cb.GetMetricsClient()
	if err != nil {
		return nil, nil, err
	}

	versionedMetrics := &metricsv1beta1.NodeMetricsList{}
	var nodes []corev1.Node
	if resourceName != "" {
		m, err := metricClient.MetricsV1beta1().NodeMetricses().Get(ctx, resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}
		versionedMetrics.Items = []metricsv1beta1.NodeMetrics{*m}

		node, err := cli.CoreV1().Nodes().Get(ctx, resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, *node)
	} else {
		options := metav1.ListOptions{}
		if len(labelSelector) > 0 {
			options.LabelSelector = labelSelector
		}

		versionedMetrics, err = metricClient.MetricsV1beta1().NodeMetricses().List(ctx, options)
		if err != nil {
			return nil, nil, err
		}

		nodeList, err := cli.CoreV1().Nodes().List(ctx, options)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, nodeList.Items...)
	}

	metrics := &metricsapi.NodeMetricsList{}
	if err = metricsv1beta1.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(versionedMetrics, metrics, nil); err != nil {
		return nil, nil, err
	}

	availableResources := make(map[string]corev1.ResourceList)
	for _, n := range nodes {
		availableResources[n.Name] = n.Status.Capacity
	}

	out := bytes.NewBuffer(make([]byte, 0))
	if err := metricsutil.NewTopCmdPrinter(out).PrintNodeMetrics(metrics.Items, availableResources, false, sortBy); err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: out.String()},
		},
		StructuredContent: out,
	}, out, nil
}
