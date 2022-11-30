package webhooks

import (
	"net/url"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewDeactivateCommand() *cobra.Command {
	return internal.NewCommand("deactivate",
		internal.WithShortDescription("Deactivate a webhook"),
		internal.WithAliases("deac"),
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}
			webhookClient, err := NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			if _, err := url.Parse(args[0]); err != nil {
				return err
			}

			_, _, err = webhookClient.ConfigsApi.DeactivateOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(), "Config deactivated")
			return nil
		}),
	)
}
