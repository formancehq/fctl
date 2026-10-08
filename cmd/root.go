package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/cmd/version"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "fctl",
		Short:         "Formance Control CLI",
		Version:       version.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(version.NewCommand())
	return root
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return NewRootCommand().ExecuteContext(ctx)
}
