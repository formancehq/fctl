package clients

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete [CLIENT_ID]",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithAliases("d", "del"),
		fctl.WithShortDescription("Delete client"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.Get(cmd)
			if err != nil {
				return err
			}

			authClient, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = authClient.ClientsApi.DeleteClient(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			fctl.Success(cmd.OutOrStdout(), "Client deleted!")
			return nil
		}),
	)
}
