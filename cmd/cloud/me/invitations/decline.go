package invitations

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewDeclineCommand() *cobra.Command {
	return internal.NewCommand("decline",
		internal.WithAliases("dec", "d"),
		internal.WithShortDescription("Decline invitation"),
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = client.DefaultApi.DeclineInvitation(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(), "Invitation declined!")
			return nil
		}),
	)
}
