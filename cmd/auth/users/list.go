package users

import (
	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
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

			client, err := internal.NewAuthClient(cmd, cfg)
			if err != nil {
				return err
			}

			listUsersResponse, _, err := client.DefaultApi.ListUsers(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			tableData := collections.Map(listUsersResponse.Data, func(o authclient.User) []string {
				return []string{
					*o.Id,
					*o.Subject,
					*o.Email,
				}
			})
			tableData = collections.Prepend(tableData, []string{"ID", "Subject", "Email"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
