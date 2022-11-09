package cmd

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func newAuthClientsShowCommand() *cobra.Command {
	return newCommand("show [CLIENT_ID]",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			authClient, err := newAuthClient(cmd)
			if err != nil {
				return err
			}

			response, _, err := authClient.DefaultApi.ReadClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}
			fctl.PrintAuthClient(cmd.OutOrStdout(), *response.Data)

			return nil
		}),
	)
}
