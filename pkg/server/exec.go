package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// RunInContainerArgs represents the arguments for the RunInContainer tool.
type RunInContainerArgs struct {
	Name      string   `json:"name" jsonschema:"Name of the Pod where the command will be executed"`
	Namespace string   `json:"namespace" jsonschema:"Namespace of the Pod where the command will be executed"`
	Container string   `json:"container" jsonschema:"The container name which execute command in the pod"`
	Command   []string `json:"command" jsonschema:"Command to execute in the Pod container"`
}

type RunInContainerResult map[string]string

func (s *Server) RunInContainer(ctx context.Context, req *mcp.CallToolRequest, args *RunInContainerArgs) (*mcp.CallToolResult, RunInContainerResult, error) {
	resourceName := args.Name
	namespace := args.Namespace
	command := args.Command
	containerName := args.Container

	slog.Info("Executing command in container", "resourceName", resourceName, "namespace", namespace, "container", containerName, "command", command)

	cli, err := s.cb.GetClient()
	if err != nil {
		return nil, nil, err
	}

	// Check if the Pod exists and is not completed
	pod, err := cli.CoreV1().Pods(namespace).Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return nil, nil, fmt.Errorf("cannot exec into a container in a completed pod, current phase is %s", pod.Status.Phase)
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
		return nil, nil, err
	}

	var stdout = bytes.NewBuffer(make([]byte, 0))
	var stderr = bytes.NewBuffer(make([]byte, 0))
	if err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: stdout, Stderr: stderr, Tty: false}); err != nil {
		return nil, nil, err
	}

	result := RunInContainerResult{
		"stdout": stdout.String(),
		"stderr": stderr.String(),
	}
	out, err := json.Marshal(result)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(out)},
		},
		StructuredContent: result,
	}, result, nil
}

// createExecutor:
// copy from
// https://github.com/kubernetes/kubernetes/blob/bd44685eadc64c8cd46a8259f027f57ba9724a85/staging/src/k8s.io/kubectl/pkg/cmd/exec/exec.go#L146-L166
func (s *Server) createExecutor(namespace, name string, podExecOptions *corev1.PodExecOptions) (remotecommand.Executor, error) {
	cli, err := s.cb.GetClient()
	if err != nil {
		return nil, err
	}

	cfg, err := s.cb.ToRawKubeConfigLoader().ClientConfig()
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
