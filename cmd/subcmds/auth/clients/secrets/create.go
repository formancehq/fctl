package secrets

import (
	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/auth/internal"
	"github.com/spf13/cobra"
)

func NewAuthClientsSecretsCreateCommand() *cobra.Command {
	return cmdbuilder.NewCommand("create [CLIENT_ID] [SECRET_NAME]",
		cmdbuilder.WithArgs(cobra.ExactArgs(2)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd, cfg)
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

			internal.PrintAuthClientSecret(cmd.OutOrStdout(), response.Data)

			return nil
		}),
	)
}
