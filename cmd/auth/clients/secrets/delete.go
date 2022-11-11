package secrets

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewAuthClientsSecretsDeleteCommand() *cobra.Command {
	return cmdbuilder.NewCommand("delete [CLIENT_ID] [SECRET_ID]",
		cmdbuilder.WithArgs(cobra.ExactArgs(2)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = authClient.DefaultApi.
				DeleteSecret(cmd.Context(), args[0], args[1]).
				Execute()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Secret deleted!")

			return nil
		}),
	)
}
