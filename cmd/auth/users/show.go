package users

import (
	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return internal2.NewCommand("show",
		internal2.WithAliases("s"),
		internal2.WithShortDescription("Show user"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			readUserResponse, _, err := client.UsersApi.ReadUser(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("ID"), *readUserResponse.Data.Id})
			tableData = append(tableData, []string{pterm.LightCyan("Membership ID"), *readUserResponse.Data.Subject})
			tableData = append(tableData, []string{pterm.LightCyan("Email"), *readUserResponse.Data.Email})

			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
