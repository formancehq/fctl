package webhooks

import (
	"net/url"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	const (
		secretFlag = "secret"
	)
	return fctl.NewCommand("create [ENDPOINT] [EVENT_TYPE1] [EVENT_TYPE2,optional]...",
		fctl.WithShortDescription("Create a new config. At least one event type is required."),
		fctl.WithAliases("cr"),
		fctl.WithArgs(cobra.MinimumNArgs(2)),
		fctl.WithStringFlag(secretFlag, "", "Bring your own webhooks signing secret. If not passed or empty, a secret is automatically generated. The format is a string of bytes of size 24, base64 encoded. (larger size after encoding)"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return errors.Wrap(err, "fctl.GetConfig")
			}

			client, err := fctl.NewStackClient(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "fctl.NewStackClient")
			}

			if _, err := url.Parse(args[0]); err != nil {
				return errors.Wrap(err, "invalid endpoint URL")
			}

			secret := fctl.GetString(cmd, secretFlag)

			res, _, err := client.WebhooksApi.InsertOneConfig(cmd.Context()).
				ConfigUser(formance.ConfigUser{
					Endpoint:   &args[0],
					EventTypes: args[1:],
					Secret:     &secret,
				}).Execute()
			if err != nil {
				return errors.Wrap(err, "inserting config")
			}

			fctl.Success(cmd.OutOrStdout(),
				"Config created successfully with ID: %s", *res.Data.Id)
			return nil
		}),
	)
}
