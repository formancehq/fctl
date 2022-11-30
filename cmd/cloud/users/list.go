package users

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal.NewCommand("list",
		internal.WithAliases("ls", "l"),
		internal.WithShortDescription("List users"),
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

			usersResponse, _, err := apiClient.DefaultApi.ListUsers(cmd.Context(), organizationID).Execute()
			if err != nil {
				return err
			}

			tableData := internal.Map(usersResponse.Data, func(i membershipclient.User) []string {
				return []string{
					i.Id,
					i.Email,
				}
			})
			tableData = internal.Prepend(tableData, []string{"ID", "Email"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
