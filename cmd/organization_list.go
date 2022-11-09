package cmd

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func newOrganizationsListCommand() *cobra.Command {
	return newCommand("list",
		withShortDescription("list organizations"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			apiClient, err := newMembershipClient(cmd)
			if err != nil {
				return err
			}

			organizations, _, err := apiClient.DefaultApi.ListOrganizations(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Organizations: ")
			for _, o := range organizations.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "-> Organization: %s\r\n", o.Id)
				internal.PrintOrganization(cmd.OutOrStdout(), o)
			}
			return nil
		}),
	)
}
