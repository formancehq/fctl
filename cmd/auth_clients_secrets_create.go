package cmd

import (
	"github.com/formancehq/auth/authclient"
	fctl "github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func newAuthClientsSecretsCreateCommand() *cobra.Command {
	return newCommand("create [CLIENT_ID] [SECRET_NAME]",
		withArgs(cobra.ExactArgs(2)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			authClient, err := newAuthClient(cmd)
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

			fctl.PrintAuthClientSecret(cmd.OutOrStdout(), response.Data)

			return nil
		}),
	)
}
