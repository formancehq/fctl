package cmd

import (
	"fmt"

	fctl "github.com/formancehq/fctl/cmd/internal"
	"github.com/pborman/indent"
	"github.com/spf13/cobra"
)

func newAuthClientsListCommand() *cobra.Command {
	return newCommand("list",
		withRunE(func(cmd *cobra.Command, args []string) error {
			authClient, err := newAuthClient(cmd)
			if err != nil {
				return err
			}

			clients, _, err := authClient.DefaultApi.ListClients(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			w := indent.New(cmd.OutOrStdout(), "\t")
			for _, client := range clients.Data {
				fmt.Fprintln(cmd.OutOrStdout(), "-")
				fctl.PrintAuthClient(w, client)
			}

			return nil
		}),
	)
}
