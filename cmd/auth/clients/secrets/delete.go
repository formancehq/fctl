package secrets

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete [CLIENT_ID] [SECRET_ID]",
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithAliases("d"),
		fctl.WithShortDescription("Delete secret"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			authClient, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = authClient.ClientsApi.
				DeleteSecret(cmd.Context(), args[0], args[1]).
				Execute()
			if err != nil {
				return err
			}

			fctl.Success(cmd.OutOrStdout(), "Secret deleted!")

			return nil
		}),
	)
}
