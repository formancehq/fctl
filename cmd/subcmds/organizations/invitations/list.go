package invitations

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

func PrintInvitation(out io.Writer, invitation membershipclient.Invitation) {
	fmt.Fprintf(out, "Email: '%s'\r\n", invitation.UserEmail)
	fmt.Fprintf(out, "CreatedAt: '%s'\r\n", invitation.CreatedAt)
	fmt.Fprintf(out, "Status: '%s'\r\n", invitation.Status)
}

func NewOrganizationsInvitationsListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("ls"),
		cmdbuilder.WithArgs(cobra.ExactArgs(0)),
		cmdbuilder.WithShortDescription("list invitations"),
		cmdbuilder.WithAliases("s"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig()
			if err != nil {
				return err
			}

			apiClient, err := membership.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			organizationID, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			listInvitationsResponse, _, err := apiClient.DefaultApi.
				ListOrganizationInvitations(cmd.Context(), organizationID).
				Execute()
			if err != nil {
				return err
			}

			for _, invitation := range listInvitationsResponse.Data {
				if invitation.Status == "ACCEPTED" {
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Invitation '%s'\r\n", invitation.Id)
				PrintInvitation(cmd.OutOrStdout(), invitation)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Invitation sent\r\n")
			return nil
		}),
	)
}
