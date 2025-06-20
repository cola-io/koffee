package server

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/metricsutil"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func (s *Server) TopPod() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		namespace := req.GetString("namespace", metav1.NamespaceAll)
		resourceName := req.GetString("name", "")
		sortBy := req.GetString("sortBy", "")
		labelSelector := req.GetString("labelSelector", "")
		fieldSelector := req.GetString("fieldSelector", "")

		slog.Info("Loading top pod argument", "namespace", namespace, "resourceName", resourceName, "sortBy", sortBy, "labelSelector", labelSelector, "fieldSelector", fieldSelector)

		metricClient, err := s.cb.GetMetricsClient()
		if err != nil {
			return nil, err
		}

		versionedMetrics := &metricsv1beta1.PodMetricsList{}
		if resourceName != "" {
			m, err := metricClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, resourceName, metav1.GetOptions{})
			if err != nil {
				return nil, err
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
				return nil, err
			}
		}

		metrics := &metricsapi.PodMetricsList{}
		if err = metricsv1beta1.Convert_v1beta1_PodMetricsList_To_metrics_PodMetricsList(versionedMetrics, metrics, nil); err != nil {
			return nil, err
		}

		out := bytes.NewBuffer(make([]byte, 0))
		if err := metricsutil.NewTopCmdPrinter(out).PrintPodMetrics(metrics.Items, true, true, false, sortBy, true); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(out.String()), nil
	}
}

func (s *Server) TopNode() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceName := req.GetString("name", "")
		sortBy := req.GetString("sortBy", "")
		labelSelector := req.GetString("labelSelector", "")

		slog.Info("Loading top node argument", "resourceName", resourceName, "sortBy", sortBy, "labelSelector", labelSelector)

		cli, err := s.cb.GetClient()
		if err != nil {
			return nil, err
		}

		metricClient, err := s.cb.GetMetricsClient()
		if err != nil {
			return nil, err
		}

		versionedMetrics := &metricsv1beta1.NodeMetricsList{}
		var nodes []corev1.Node
		if resourceName != "" {
			m, err := metricClient.MetricsV1beta1().NodeMetricses().Get(ctx, resourceName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			versionedMetrics.Items = []metricsv1beta1.NodeMetrics{*m}

			node, err := cli.CoreV1().Nodes().Get(ctx, resourceName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, *node)
		} else {
			options := metav1.ListOptions{}
			if len(labelSelector) > 0 {
				options.LabelSelector = labelSelector
			}

			versionedMetrics, err = metricClient.MetricsV1beta1().NodeMetricses().List(ctx, options)
			if err != nil {
				return nil, err
			}

			nodeList, err := cli.CoreV1().Nodes().List(ctx, options)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, nodeList.Items...)
		}

		metrics := &metricsapi.NodeMetricsList{}
		if err = metricsv1beta1.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(versionedMetrics, metrics, nil); err != nil {
			return nil, err
		}

		availableResources := make(map[string]corev1.ResourceList)
		for _, n := range nodes {
			availableResources[n.Name] = n.Status.Capacity
		}

		out := bytes.NewBuffer(make([]byte, 0))
		if err := metricsutil.NewTopCmdPrinter(out).PrintNodeMetrics(metrics.Items, availableResources, false, sortBy); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(out.String()), nil
	}
}
