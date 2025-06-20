package options

import (
	"errors"
	"fmt"
	"os"

	cliflag "k8s.io/component-base/cli/flag"

	"cola.io/koffee/pkg/version"
)

const (
	StdioTransport = "stdio"
	SSETransport   = "sse"
)

// Options defines all options for the koffee.
type Options struct {
	Transport  string
	Port       int
	Kubeconfig string
	Verbose    int
	Version    bool
}

// NewOptions returns a new Options object.
func NewOptions() *Options {
	return &Options{
		Transport: StdioTransport,
		Verbose:   0,
		Port:      8888,
	}
}

func (o *Options) AddFlags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("koffee")
	fs.StringVarP(&o.Kubeconfig, "kubeconfig", "k", "", "Path to Kubernetes configuration file (uses default config if not specified)")
	fs.StringVarP(&o.Transport, "transport", "t", o.Transport, "Transport protocol to use (stdio, sse)")
	fs.IntVarP(&o.Port, "port", "p", o.Port, "Port to use for communicating with server, required when using --transport=sse and must be between 1 and 65535")
	fs.IntVarP(&o.Verbose, "v", "v", o.Verbose, "Setting the slog level, default is info level")
	fs.BoolVarP(&o.Version, "version", "V", o.Version, "Print version information and quits")
	return
}

func (o *Options) Validate() error {
	if o.Transport != StdioTransport && o.Transport != SSETransport {
		return errors.New("--transport must be one of (stdio, sse)")
	}

	if o.Transport == "sse" && (o.Port < 1 || o.Port > 65535) {
		return errors.New("--port is required when using --transport=sse and must be between 1 and 65535")
	}
	return nil
}

func (o *Options) PrintAndExitIfRequested() {
	if o.Version {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", version.Get().Pretty())
		os.Exit(0)
	}
}
