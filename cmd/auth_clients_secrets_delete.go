package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAuthClientsSecretsDeleteCommand() *cobra.Command {
	return newCommand("delete [CLIENT_ID] [SECRET_ID]",
		withArgs(cobra.ExactArgs(2)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			authClient, err := newAuthClient(cmd)
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
