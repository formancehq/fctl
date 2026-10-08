package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd/cloudtools"
	"github.com/formancehq/fctl/v4/cmd/connections"
	"github.com/formancehq/fctl/v4/cmd/login"
	"github.com/formancehq/fctl/v4/cmd/version"
	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/plugins/auth"
	cloudplugin "github.com/formancehq/fctl/v4/plugins/cloud"
	"github.com/formancehq/fctl/v4/plugins/ledger"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "fctl",
		Short:         "Formance Control CLI",
		Version:       version.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Example:       "  fctl login\n  fctl cloud stack list --organization ORGANIZATION_ID\n  fctl ledger list --organization ORGANIZATION_ID --stack STACK_ID\n  fctl ledger list -o json",
	}
	settings := &connection.Settings{}
	settings.Bind(root)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		return command.ConfigureOutput(cmd, settings.Output, settings.Color)
	}
	command.InstallHelp(root, &settings.Color)
	root.AddGroup(&cobra.Group{ID: "cloud", Title: "Cloud:"}, &cobra.Group{ID: "modules", Title: "Modules:"}, &cobra.Group{ID: "connections", Title: "Connections:"})
	resolve := func(ctx context.Context, service string, request pluginsdk.ExecuteRequest) (*api.Client, error) {
		if service == "cloud-apps" {
			return settings.ApplicationClient(ctx, root, request.Flags["deploy-app-alias"])
		}
		return settings.Client(ctx, root, service)
	}
	root.AddCommand(version.NewCommand(), connections.NewCommand(settings), login.NewCommand(settings), login.NewLogoutCommand(settings))
	registry := &plugin.Registry{}
	for _, factory := range []plugin.Factory{cloudplugin.New, auth.New, ledger.New} {
		if err := registry.Register(context.Background(), factory(nil), factory); err != nil {
			panic(err) // Embedded metadata is a build-time invariant, never user input.
		}
	}
	if err := plugin.NewCommandWithRequest(registry, resolve).AddTo(root); err != nil {
		panic(err)
	}
	if err := cloudtools.AddTo(root, settings); err != nil {
		panic(err)
	}
	for _, cmd := range root.Commands() {
		switch cmd.Name() {
		case "cloud":
			cmd.GroupID = "cloud"
		case "auth", "ledger":
			cmd.GroupID = "modules"
		case "connections", "login", "logout":
			cmd.GroupID = "connections"
		}
	}
	return root
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return NewRootCommand().ExecuteContext(ctx)
}
