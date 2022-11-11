package cmd

import (
	"fmt"
	"io"

	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

func PrintInvitation(out io.Writer, invitation membershipclient.Invitation) {
	fmt.Fprintf(out, "Email: '%s'\r\n", invitation.UserEmail)
	fmt.Fprintf(out, "CreatedAt: '%s'\r\n", invitation.CreatedAt)
	fmt.Fprintf(out, "Status: '%s'\r\n", invitation.Status)
}

func newOrganizationsInvitationsListCommand() *cobra.Command {
	return newCommand("list",
		withAliases("ls"),
		withArgs(cobra.ExactArgs(0)),
		withShortDescription("list invitations"),
		withAliases("s"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			config, err := getConfig()
			if err != nil {
				return err
			}

			apiClient, err := newMembershipClient(cmd, config)
			if err != nil {
				return err
			}

			organizationID, err := resolveOrganizationID(cmd, config)
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
