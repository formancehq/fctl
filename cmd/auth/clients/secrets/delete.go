package secrets

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return internal2.NewCommand("delete [CLIENT_ID] [SECRET_ID]",
		internal2.WithArgs(cobra.ExactArgs(2)),
		internal2.WithAliases("d"),
		internal2.WithShortDescription("Delete secret"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			authClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = authClient.ClientsApi.
				DeleteSecret(cmd.Context(), args[0], args[1]).
				Execute()
			if err != nil {
				return err
			}

			internal2.Success(cmd.OutOrStdout(), "Secret deleted!")

			return nil
		}),
	)
}
