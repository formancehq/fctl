package secrets

import (
	"github.com/formancehq/auth/authclient"
	internal2 "github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewAuthClientsSecretsCreateCommand() *cobra.Command {
	return cmdbuilder.NewCommand("create [CLIENT_ID] [SECRET_NAME]",
		cmdbuilder.WithArgs(cobra.ExactArgs(2)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authClient, err := internal2.NewAuthClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := authClient.DefaultApi.
				CreateSecret(cmd.Context(), args[0]).
				Body(authclient.SecretOptions{
					Name:     args[1],
					Metadata: nil,
				}).
				Execute()
			if err != nil {
				return err
			}

			internal2.PrintAuthClientSecret(cmd.OutOrStdout(), response.Data)

			return nil
		}),
	)
}
