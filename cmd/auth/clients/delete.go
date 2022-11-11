package clients

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewAuthClientsDeleteCommand() *cobra.Command {
	return cmdbuilder.NewCommand("delete [CLIENT_ID]",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = authClient.DefaultApi.DeleteClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Client deleted!")
			return nil
		}),
	)
}
