package webhooks

import (
	"net/url"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	const (
		secretFlag = "secret"
	)
	return fctl.NewCommand("create",
		fctl.WithShortDescription("Create a new config"),
		fctl.WithAliases("cr"),
		fctl.WithArgs(cobra.MinimumNArgs(2)),
		fctl.WithStringFlag(secretFlag, "", "Webhooks signing secret"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			client, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			if _, err := url.Parse(args[0]); err != nil {
				return err
			}

			secret := fctl.GetString(cmd, secretFlag)

			res, _, err := client.WebhooksApi.InsertOneConfig(cmd.Context()).
				ConfigUser(formance.ConfigUser{
					Endpoint:   &args[0],
					EventTypes: args[1:],
					Secret:     &secret,
				}).Execute()
			if err != nil {
				return err
			}

			fctl.Success(cmd.OutOrStdout(),
				"Config created successfully with ID: %s", *res.Data.Id)
			return nil
		}),
	)
}
