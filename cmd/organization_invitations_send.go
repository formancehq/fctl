package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newOrganizationsInvitationsSendCommand() *cobra.Command {
	return newCommand("send",
		withArgs(cobra.ExactArgs(1)),
		withShortDescription("invite on organization by email"),
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
