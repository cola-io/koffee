package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func (s *Server) GetPodLogs() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceName, err := req.RequireString("name")
		if err != nil {
			return nil, err
		}

		namespace, err := req.RequireString("namespace")
		if err != nil {
			return nil, err
		}

		// If containerName is empty, the default container will be used by Kubernetes
		containerName := req.GetString("container", "")
		tailLines := req.GetInt("tail", 50)

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
			if err := podLogs.Close(); err != nil {
				slog.Error("Failed to close pod logs", "err", err)
			}
		}()

		buf := bytes.NewBuffer(make([]byte, 0))
		if _, err = io.Copy(buf, podLogs); err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(buf.String()), nil
	}
}
