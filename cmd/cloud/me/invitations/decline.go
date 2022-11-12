package invitations

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
)

func NewDeclineCommand() *cobra.Command {
	return cmdbuilder.NewCommand("decline",
		cmdbuilder.WithAliases("dec", "d"),
		cmdbuilder.WithShortDescription("Decline invitation"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			client, err := membership.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			_, err = client.DefaultApi.DeclineInvitation(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Invitation declined!")
			return nil
		}),
	)
}
