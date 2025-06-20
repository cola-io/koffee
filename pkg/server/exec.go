package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

func (s *Server) RunInContainer() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceName, err := req.RequireString("name")
		if err != nil {
			return nil, err
		}

		namespace, err := req.RequireString("namespace")
		if err != nil {
			return nil, err
		}

		command, err := req.RequireStringSlice("command")
		if err != nil {
			return nil, err
		}
		containerName := req.GetString("container", "")

		slog.Info("Executing command in container", "resourceName", resourceName, "namespace", namespace, "container", containerName, "command", command)

		cli, err := s.cb.GetClient()
		if err != nil {
			return nil, err
		}

		// Check if the Pod exists and is not completed
		pod, err := cli.CoreV1().Pods(namespace).Get(ctx, resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return nil, fmt.Errorf("cannot exec into a container in a completed pod, current phase is %s", pod.Status.Phase)
		}

		executor, err := s.createExecutor(namespace, resourceName, &corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		})
		if err != nil {
			return nil, err
		}

		var stdout = bytes.NewBuffer(make([]byte, 0))
		var stderr = bytes.NewBuffer(make([]byte, 0))
		if err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: stdout, Stderr: stderr, Tty: false}); err != nil {
			return nil, err
		}

		resp, err := json.Marshal(map[string]string{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
		})
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(resp)), nil
	}
}

// createExecutor:
// copy from
// https://github.com/kubernetes/kubernetes/blob/bd44685eadc64c8cd46a8259f027f57ba9724a85/staging/src/k8s.io/kubectl/pkg/cmd/exec/exec.go#L146-L166
func (s *Server) createExecutor(namespace, name string, podExecOptions *corev1.PodExecOptions) (remotecommand.Executor, error) {
	cli, err := s.cb.GetClient()
	if err != nil {
		return nil, err
	}

	cfg, err := s.cb.LoadRESTConfig()
	if err != nil {
		return nil, err
	}

	req := cli.CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(name).
		SubResource("exec").
		VersionedParams(podExecOptions, scheme.ParameterCodec)

	spdyExec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	webSocketExec, err := remotecommand.NewWebSocketExecutor(cfg, "GET", req.URL().String())
	if err != nil {
		return nil, err
	}

	return remotecommand.NewFallbackExecutor(webSocketExec, spdyExec, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
}
