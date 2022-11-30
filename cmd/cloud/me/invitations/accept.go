package invitations

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewAcceptCommand() *cobra.Command {
	return internal.NewCommand("accept",
		internal.WithAliases("a"),
		internal.WithShortDescription("Accept invitation"),
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

			_, err = client.DefaultApi.AcceptInvitation(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(), "Invitation accepted!")
			return nil
		}),
	)
}
