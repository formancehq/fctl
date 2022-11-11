package clients

import (
	"fmt"

	internal2 "github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pborman/indent"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authClient, err := internal2.NewAuthClient(cmd.Context(), cfg)
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
				internal2.PrintAuthClient(w, client)
			}

			return nil
		}),
	)
}
