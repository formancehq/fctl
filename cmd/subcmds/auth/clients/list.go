package clients

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/auth/internal"
	"github.com/pborman/indent"
	"github.com/spf13/cobra"
)

func NewAuthClientsListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd, cfg)
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
				internal.PrintAuthClient(w, client)
			}

			return nil
		}),
	)
}
