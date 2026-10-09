package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd/cloudtools"
	"github.com/formancehq/fctl/v4/cmd/factory"
	"github.com/formancehq/fctl/v4/cmd/login"
	"github.com/formancehq/fctl/v4/cmd/plugins"
	"github.com/formancehq/fctl/v4/cmd/profiles"
	"github.com/formancehq/fctl/v4/cmd/version"
	"github.com/formancehq/fctl/v4/internal/command"
	"github.com/formancehq/fctl/v4/internal/connection"
)

func NewRootCommand() *cobra.Command {
	return newRootCommand(context.Background(), nil)
}

// NewRootCommandWithArgs selects installed plugin metadata before Cobra parses
// service flags. Help and completion read cached metadata without starting a
// plugin process. Auth and Ledger are external-only; an unprepared target exposes an
// installation guide instead of an embedded implementation.
func NewRootCommandWithArgs(ctx context.Context, args []string) *cobra.Command {
	return newRootCommand(ctx, args)
}

//nolint:contextcheck // Command construction attaches callbacks; execution uses Cobra's context.
func newRootCommand(ctx context.Context, args []string) *cobra.Command {
	root := &cobra.Command{
		Use:           "fctl",
		Short:         "Formance Control CLI",
		Version:       version.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Example:       "  fctl login\n  fctl cloud stack list --organization ORGANIZATION_ID\n  fctl auth clients list --organization ORGANIZATION_ID --stack STACK_ID",
	}
	root.SetContext(ctx)
	settings := &connection.Settings{}
	settings.Bind(root)
	root.PersistentFlags().Bool("no-input", false, "Disable interactive forms and selections (FCTL_NO_INPUT, CI)")
	services := factory.Prepare(ctx, root, settings, args)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := services.PreparationError(cmd); err != nil {
			return err
		}
		return command.ConfigureOutput(cmd, settings.Output, settings.Color)
	}
	command.InstallHelp(root, &settings.Color)
	root.AddGroup(&cobra.Group{ID: "cloud", Title: "Cloud:"}, &cobra.Group{ID: "modules", Title: "Modules:"}, &cobra.Group{ID: "profiles", Title: "Profiles:"}, &cobra.Group{ID: "plugins", Title: "Plugins:"})
	root.AddCommand(version.NewCommand(), profiles.NewCommand(settings), login.NewCommand(settings), login.NewLogoutCommand(settings))
	root.AddCommand(plugins.NewCommand(settings))
	if err := services.AddTo(root, settings); err != nil {
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
		case "profiles", "login", "logout":
			cmd.GroupID = "profiles"
		case "plugins":
			cmd.GroupID = "plugins"
		}
	}
	return root
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return NewRootCommandWithArgs(ctx, os.Args[1:]).ExecuteContext(ctx)
}
