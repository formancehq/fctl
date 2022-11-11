package clients

import (
	internal2 "github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show [CLIENT_ID]",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithShortDescription("Show client"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authClient, err := internal2.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			response, _, err := authClient.DefaultApi.ReadClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}
			internal2.PrintAuthClient(cmd.OutOrStdout(), *response.Data)

			return nil
		}),
	)
}
