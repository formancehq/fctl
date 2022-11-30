package organizations

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal.NewCommand("list",
		internal.WithAliases("ls", "l"),
		internal.WithShortDescription("List organizations"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			apiClient, err := internal.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			organizations, _, err := apiClient.DefaultApi.ListOrganizations(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			tableData := internal.Map(organizations.Data, func(o membershipclient.Organization) []string {
				return []string{
					o.Id, o.Name, o.OwnerId,
				}
			})
			tableData = internal.Prepend(tableData, []string{"ID", "Name", "Owner ID"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
