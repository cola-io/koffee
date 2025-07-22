package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

// GetPodLogsArgs represents the arguments for the GetPodLogs tool.
type GetPodLogsArgs struct {
	Name      string `json:"name" mcp:"The specified pod name"`
	Namespace string `json:"namespace" mcp:"The namespace of the pod"`
	Container string `json:"container" mcp:"Get the logs of this container in the pod"`
	TailLines int    `json:"tail" mcp:"Lines of recent log file to display"`
}

func (s *Server) GetPodLogs(ctx context.Context, session *mcp.ServerSession, req *mcp.CallToolParamsFor[GetPodLogsArgs]) (*mcp.CallToolResultFor[*bytes.Buffer], error) {
	resourceName := req.Arguments.Name
	namespace := req.Arguments.Namespace
	// If containerName is empty, the default container will be used by Kubernetes
	containerName := req.Arguments.Container
	tailLines := req.Arguments.TailLines

	slog.Info("Loading arguments", "resourceName", resourceName, "namespace", namespace, "container", containerName, "tailLines", tailLines)

	cli, err := s.cb.GetClient()
	if err != nil {
		return nil, err
	}

	podLogs, err := cli.CoreV1().Pods(namespace).GetLogs(resourceName, &corev1.PodLogOptions{
		TailLines: ptr.To(int64(tailLines)),
		Container: containerName,
	}).Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = podLogs.Close(); err != nil {
			slog.Error("Failed to close pod logs", "err", err)
		}
	}()

	buf := bytes.NewBuffer(make([]byte, 0))
	if _, err = io.Copy(buf, podLogs); err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[*bytes.Buffer]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "get pod logs successfully"},
		},
		StructuredContent: buf,
	}, nil
}
