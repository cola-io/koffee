package options

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Addr       string
	Kubeconfig string
	Verbose    int
	Version    bool
}

// NewOptions returns a new Options object.
func NewOptions() *Options {
	return &Options{
		Transport: StdioTransport,
		Addr:      ":8888",
	}
}

func (o *Options) AddFlags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("koffee")
	fs.StringVarP(&o.Kubeconfig, "kubeconfig", "k", "", "Path to Kubernetes configuration file (uses default config if not specified)")
	fs.StringVarP(&o.Transport, "transport", "t", o.Transport, "Transport protocol to use (stdio, sse)")
	fs.StringVar(&o.Addr, "addr", o.Addr, "Port to use for communicating with server, required when using --transport=sse")
	fs.IntVarP(&o.Verbose, "v", "v", o.Verbose, "Setting the slog level, default is info level")
	fs.BoolVarP(&o.Version, "version", "V", o.Version, "Print version information and quits")
	return
}

func (o *Options) Validate() error {
	if o.Transport != StdioTransport && o.Transport != SSETransport {
		return errors.New("--transport must be one of (stdio, sse)")
	}

	if o.Transport == "sse" && o.Addr == "" {
		return errors.New("--port is required when using --transport=sse")
	}
	return nil
}

func (o *Options) Complete() error {
	slog.SetDefault(slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource:   true,
			Level:       slog.Level(o.Verbose),
			ReplaceAttr: makeReplaceAttrFunc(),
		}),
	))
	return nil
}

func (o *Options) PrintAndExitIfRequested() {
	if o.Version {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", version.Get().Pretty())
		os.Exit(0)
	}
}

func makeReplaceAttrFunc() func(groups []string, a slog.Attr) slog.Attr {
	return func(_ []string, attr slog.Attr) slog.Attr {
		switch attr.Key {
		case slog.TimeKey:
			attr.Value = slog.StringValue(attr.Value.Any().(time.Time).Format("2006-01-02T15:04:05.999"))
		case slog.SourceKey:
			src := attr.Value.Any().(*slog.Source)
			attr.Value = slog.StringValue(strings.Join([]string{
				filepath.Base(src.File),
				fmt.Sprintf("%d", src.Line),
			}, ":"))
		}
		return attr
	}
}
