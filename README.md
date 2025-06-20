A tool for implementing the Model Context Protocol server. It provides a simple way to interact with Kubernetes resources.

# Features
- List local kube context, like `kubectl config get-contexts`
- Switch the kube context, like `kubectl config use-context <context>`
- Get the cluster version, like `kubectl get --raw /version`
- Get the cluster resource, like `kubectl api-resources`
- Get the resource detail info, like `kubectl get <kind> <name> -n <namespace> -oyaml`
- Apply resource with the specified manifest file, like `kubectl apply -f <file>`
- Delete resource with the specified kind, name and namespace, like `kubectl delete <kind> <name> -n <namespace>`
- Logs pod for the specified pod and container, like `kubectl logs <pod> -n <namespace>`
- Run command in the specified pod and container, like `kubectl exec <pod> -n <namespace> -c <container> -- <command>`

# Getting start

### Build local binary
```bash
# execute the following command to build the local binary.
~ » make all
~ » bin/koffee -h
A tool for implementing the Model Context Protocol server. It provides a simple way to interact with Kubernetes resources.

Usage:
  koffee [flags]

Koffee flags:

  -k, --kubeconfig string
                Path to Kubernetes configuration file (uses default config if not specified)
  -p, --port int
                Port to use for communicating with server, required when using --transport=sse and must be between 1 and 65535 (default 8888)
  -t, --transport string
                Transport protocol to use (stdio, sse) (default "stdio")
  -v, --v int
                Setting the slog level, default is info level
  -V, --version
                Print version information and quits

Global flags:

  -h, --help
                help for koffee
```

### Build docker image
```bash
# execute the following command to build the docker image.
make docker-build
```

# Configurations
## STDIO Mode
In stdio mode, koffee communicates with the client through standard input/output streams.

```json
# Run in stdio mode, it is default mode.
"mcp": {
  "servers": {
    "Kubernetes": {
      "command": "/path/to/koffee",
      "args": [
        "--kubeconfig",
        "/path/to/kubeconfig"
      ]
    }
  }
}
```

## SSE Mode
In SSE mode, koffee communicates with the client through Server-Sent Events.

```bash
# Run in SSE mode.
/path/to/koffee --kubeconfig /path/to/kubeconfig --transport sse --port 8888
```

```json
# Run in sse mode.
"mcp": {
  "servers": {
    "Kubernetes": {
      "url": "http://localhost:8888/sse",
      "args": []
    }
  }
}
```

# Usage

If you use VS Code as the MCP client, you can refer to the introduction in this document, [VS Code MCP Introduction](https://code.visualstudio.com/blogs/2025/04/07/agentMode).

## List local kube context
> todo

## Switch the kube context
> todo

## Get the cluster version
> todo

## Get the cluster resource
> todo

## Get the resource detail info
> todo

## Apply the specified manifest into cluster
> todo

## Delete the specified resource
> todo

## pod for the specified pod and container
> todo

## Run command in the specified pod and container
> todo
