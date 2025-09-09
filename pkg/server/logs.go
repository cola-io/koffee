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
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Container string `json:"container"`
	TailLines int    `json:"tail"`
}

func (s *Server) GetPodLogs(ctx context.Context, req *mcp.CallToolRequest, args *GetPodLogsArgs) (*mcp.CallToolResult, *bytes.Buffer, error) {
	resourceName := args.Name
	namespace := args.Namespace
	// If containerName is empty, the default container will be used by Kubernetes
	containerName := args.Container
	tailLines := args.TailLines

	slog.Info("Loading arguments", "resourceName", resourceName, "namespace", namespace, "container", containerName, "tailLines", tailLines)

	cli, err := s.cb.GetClient()
	if err != nil {
		return nil, nil, err
	}

	podLogs, err := cli.CoreV1().Pods(namespace).GetLogs(resourceName, &corev1.PodLogOptions{
		TailLines: ptr.To(int64(tailLines)),
		Container: containerName,
	}).Stream(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err = podLogs.Close(); err != nil {
			slog.Error("Failed to close pod logs", "err", err)
		}
	}()

	buf := bytes.NewBuffer(make([]byte, 0))
	if _, err = io.Copy(buf, podLogs); err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: buf.String()},
		},
		StructuredContent: buf,
	}, buf, nil
}
