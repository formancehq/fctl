package webhooks

import (
	"net/url"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	const (
		secretFlag = "secret"
	)
	return internal.NewCommand("create",
		internal.WithShortDescription("Create a new config"),
		internal.WithAliases("cr"),
		internal.WithArgs(cobra.MinimumNArgs(2)),
		internal.WithStringFlag(secretFlag, "", "Webhooks signing secret"),
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

			secret := internal.GetString(cmd, secretFlag)

			res, _, err := client.WebhooksApi.InsertOneConfig(cmd.Context()).
				ConfigUser(formance.ConfigUser{
					Endpoint:   &args[0],
					EventTypes: args[1:],
					Secret:     &secret,
				}).Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(),
				"Config created successfully with endpoint: %s", *res.Data.Endpoint)
			return nil
		}),
	)
}
