package clients

import (
	"strings"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal2.NewCommand("list",
		internal2.WithAliases("ls", "l"),
		internal2.WithShortDescription("List clients"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			authClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			clients, _, err := authClient.ClientsApi.ListClients(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			tableData := internal2.Map(clients.Data, func(o formance.Client) []string {
				return []string{
					o.Id,
					o.Name,
					func() string {
						if o.Description == nil {
							return ""
						}
						return ""
					}(),
					strings.Join(o.Scopes, ","),
					internal2.BoolPointerToString(o.Public),
				}
			})
			tableData = internal2.Prepend(tableData, []string{"ID", "Name", "Description", "Scopes", "Public"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
