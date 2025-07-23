package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/term"

	"cola.io/koffee/cmd/app/options"
	"cola.io/koffee/pkg/server"
	"cola.io/koffee/pkg/signals"
	"cola.io/koffee/pkg/version"
)

// NewCommand returns a new koffee command.
func NewCommand() *cobra.Command {
	opts := options.NewOptions()
	cmd := &cobra.Command{
		Use:   version.Get().Module,
		Short: "A Kubernetes MCP Tools",
		Long:  "A tool for implementing the Model Context Protocol server. It provides a simple way to interact with Kubernetes resources.",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PrintAndExitIfRequested()
			if err := opts.Validate(); err != nil {
				return err
			}

			if err := opts.Complete(); err != nil {
				return err
			}
			return runCommand(signals.SetupSignalHandler(), opts)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}

	fs := cmd.Flags()
	namedFlagSets := opts.AddFlags()
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name(), logs.SkipLoggingConfigurationFlags())
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cliflag.SetUsageAndHelpFunc(cmd, namedFlagSets, cols)
	return cmd
}

func runCommand(ctx context.Context, opts *options.Options) error {
	svr := server.NewServer(
		opts.Kubeconfig,
		server.WithTransport(opts.Transport),
		server.WithAddr(opts.Addr),
	)
	return svr.Start(ctx)
}
