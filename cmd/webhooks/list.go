package webhooks

import (
	"strings"
	"time"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
	webhookclient "github.com/formancehq/webhooks/client"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithShortDescription("List configs"),
		cmdbuilder.WithAliases("ls", "l"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}
			webhookClient, err := newWebhookClient(cmd, cfg)
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
					collections.Prepend(
						collections.Map(response.Cursor.Data, func(src webhookclient.Config) []string {
							return []string{
								*src.Id,
								src.CreatedAt.Format(time.RFC3339),
								cmdbuilder.StringPointerToString(src.Secret),
								*src.Endpoint,
								cmdbuilder.BoolPointerToString(src.Active),
								strings.Join(src.EventTypes, ","),
							}
						}),
						[]string{"ID", "Created at", "Secret", "Endpoint", "Active", "Event types"},
					),
				).Render()
		}),
	)
}
