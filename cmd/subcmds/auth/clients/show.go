package clients

import (
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/auth/internal"
	"github.com/spf13/cobra"
)

func NewAuthClientsShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show [CLIENT_ID]",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := authClient.DefaultApi.ReadClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}
			internal.PrintAuthClient(cmd.OutOrStdout(), *response.Data)

			return nil
		}),
	)
}
