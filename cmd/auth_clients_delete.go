package cmd

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func newAuthClientsDeleteCommand() *cobra.Command {
	return newCommand("delete [CLIENT_ID]",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {
			authClient, err := fctl.NewAuthClientFromContext(cmd.Context())
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
