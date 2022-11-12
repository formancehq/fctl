package users

import (
	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show",
		cmdbuilder.WithAliases("s"),
		cmdbuilder.WithShortDescription("Show user"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			client, err := internal.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			readUserResponse, _, err := client.DefaultApi.ReadUser(cmd.Context(), args[0]).Execute()
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
