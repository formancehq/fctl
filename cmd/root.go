package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd/connections"
	"github.com/formancehq/fctl/v4/cmd/login"
	"github.com/formancehq/fctl/v4/cmd/version"
	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/plugins/auth"
	"github.com/formancehq/fctl/v4/plugins/ledger"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "fctl",
		Short:         "Formance Control CLI",
		Version:       version.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	settings := &connection.Settings{}
	settings.Bind(root)
	resolve := func(ctx context.Context, service string) (*api.Client, error) {
		return settings.Client(ctx, root, service)
	}
	root.AddCommand(version.NewCommand(), connections.NewCommand(settings), login.NewCommand(settings), login.NewLogoutCommand(settings))
	registry := &plugin.Registry{}
	for _, factory := range []plugin.Factory{auth.New, ledger.New} {
		if err := registry.Register(context.Background(), factory(nil), factory); err != nil {
			panic(err) // Embedded metadata is a build-time invariant, never user input.
		}
	}
	if err := plugin.NewCommand(registry, resolve).AddTo(root); err != nil {
		panic(err)
	}
	return root
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return NewRootCommand().ExecuteContext(ctx)
}
