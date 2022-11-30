package users

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return internal.NewCommand("show",
		internal.WithAliases("s"),
		internal.WithShortDescription("Show user by id"),
		internal.WithArgs(cobra.ExactArgs(1)),
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

			userResponse, _, err := apiClient.DefaultApi.ReadUser(cmd.Context(), organizationID, args[0]).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("ID"), userResponse.Data.Id})
			tableData = append(tableData, []string{pterm.LightCyan("Email"), userResponse.Data.Email})

			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
