package clients

import (
	"strings"

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
		cmdbuilder.WithShortDescription("List clients"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			authClient, err := internal.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			clients, _, err := authClient.DefaultApi.ListClients(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			tableData := collections.Map(clients.Data, func(o authclient.Client) []string {
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
					cmdbuilder.BoolPointerToString(o.Public),
				}
			})
			tableData = collections.Prepend(tableData, []string{"ID", "Name", "Description", "Scopes", "Public"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
