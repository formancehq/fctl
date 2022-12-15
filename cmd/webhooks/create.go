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
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.MinimumNArgs(2)),
		fctl.WithStringFlag(secretFlag, "", "Webhooks signing secret"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			organizationID, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := fctl.ResolveStack(cmd, cfg, organizationID)
			if err != nil {
				return err
			}

			if !fctl.CheckStackApprobation(cmd, stack, "You are about to create a webhook") {
				return fctl.ErrMissingApproval
			}

			client, err := fctl.NewStackClient(cmd, cfg, stack)
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
