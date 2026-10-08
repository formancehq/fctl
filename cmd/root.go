package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd/auth"
	"github.com/formancehq/fctl/v4/cmd/connections"
	"github.com/formancehq/fctl/v4/cmd/ledger"
	"github.com/formancehq/fctl/v4/cmd/login"
	"github.com/formancehq/fctl/v4/cmd/version"
	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
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
	runtime := command.Runtime{Client: func(ctx context.Context, service string) (*api.Client, error) {
		return settings.Client(ctx, root, service)
	}}
	// Modules receive the same connection boundary and register their own commands.
	for _, module := range []func(command.Runtime) *cobra.Command{auth.NewCommand, ledger.NewCommand} {
		root.AddCommand(module(runtime))
	}
	root.AddCommand(version.NewCommand(), connections.NewCommand(settings), login.NewCommand(settings), login.NewLogoutCommand(settings))
	return root
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return NewRootCommand().ExecuteContext(ctx)
}
