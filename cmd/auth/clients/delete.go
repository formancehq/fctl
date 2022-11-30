package clients

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return internal2.NewCommand("delete [CLIENT_ID]",
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithAliases("d", "del"),
		internal2.WithShortDescription("Delete client"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			authClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = authClient.ClientsApi.DeleteClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			internal2.Success(cmd.OutOrStdout(), "Client deleted!")
			return nil
		}),
	)
}
