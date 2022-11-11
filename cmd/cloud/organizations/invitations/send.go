package invitations

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
)

func NewSendCommand() *cobra.Command {
	return cmdbuilder.NewCommand("send",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithShortDescription("Invite a user by email"),
		cmdbuilder.WithAliases("s"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			apiClient, err := membership.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			organizationID, err := cmdbuilder.ResolveOrganizationID(cmd.Context(), cfg)
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

			fmt.Fprintf(cmd.OutOrStdout(), "Invitation sent\r\n")
			return nil
		}),
	)
}
