package webhooks

import (
	"strings"
	"time"

	"github.com/formancehq/fctl/cmd/internal"
	webhookclient "github.com/formancehq/webhooks/client"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal.NewCommand("list",
		internal.WithShortDescription("List configs"),
		internal.WithAliases("ls", "l"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}
			webhookClient, err := NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			response, _, err := webhookClient.ConfigsApi.GetManyConfigs(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			return pterm.DefaultTable.
				WithHasHeader(true).
				WithWriter(cmd.OutOrStdout()).
				WithData(
					internal.Prepend(
						internal.Map(response.Cursor.Data, func(src webhookclient.Config) []string {
							return []string{
								*src.Id,
								src.CreatedAt.Format(time.RFC3339),
								internal.StringPointerToString(src.Secret),
								*src.Endpoint,
								internal.BoolPointerToString(src.Active),
								strings.Join(src.EventTypes, ","),
							}
						}),
						[]string{"ID", "Created at", "Secret", "Endpoint", "Active", "Event types"},
					),
				).Render()
		}),
	)
}
