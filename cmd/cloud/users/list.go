package users

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("ls", "l"),
		cmdbuilder.WithShortDescription("List users"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}

			apiClient, err := config.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			organizationID, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			usersResponse, _, err := apiClient.DefaultApi.ListUsers(cmd.Context(), organizationID).Execute()
			if err != nil {
				return err
			}

			tableData := collections.Map(usersResponse.Data, func(i membershipclient.User) []string {
				return []string{
					i.Id,
					i.Email,
				}
			})
			tableData = collections.Prepend(tableData, []string{"ID", "Email"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
