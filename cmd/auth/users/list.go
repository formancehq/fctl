package users

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal2.NewCommand("list",
		internal2.WithAliases("ls", "l"),
		internal2.WithShortDescription("List users"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			listUsersResponse, _, err := client.UsersApi.ListUsers(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			tableData := internal2.Map(listUsersResponse.Data, func(o formance.User) []string {
				return []string{
					*o.Id,
					*o.Subject,
					*o.Email,
				}
			})
			tableData = internal2.Prepend(tableData, []string{"ID", "Subject", "Email"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
