package invitations

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewSendCommand() *cobra.Command {
	return internal.NewCommand("send",
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithShortDescription("Invite a user by email"),
		internal.WithAliases("s"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			apiClient, err := internal.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			organizationID, err := internal.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			_, _, err = apiClient.DefaultApi.
				CreateInvitation(cmd.Context(), organizationID).
				Email(args[0]).
				Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(), "Invitation sent")
			return nil
		}),
	)
}
