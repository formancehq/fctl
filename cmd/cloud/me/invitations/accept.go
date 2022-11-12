package invitations

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewAcceptCommand() *cobra.Command {
	return cmdbuilder.NewCommand("accept",
		cmdbuilder.WithAliases("a"),
		cmdbuilder.WithShortDescription("Accept invitation"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			client, err := config.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			_, err = client.DefaultApi.AcceptInvitation(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Invitation accepted!")
			return nil
		}),
	)
}
