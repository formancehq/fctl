package webhooks

import (
	"net/url"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewActivateCommand() *cobra.Command {
	return internal.NewCommand("activate",
		internal.WithShortDescription("Activate one config"),
		internal.WithAliases("ac", "a"),
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			client, err := internal.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			if _, err := url.Parse(args[0]); err != nil {
				return err
			}

			_, _, err = client.WebhooksApi.ActivateOneConfig(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(), "Config activated successfully")
			return nil
		}),
	)
}
