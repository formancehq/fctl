package webhooks

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/config"
	webhookclient "github.com/formancehq/webhooks/client"
	"github.com/spf13/cobra"
)

func newWebhookClient(cmd *cobra.Command, cfg *config.Config) (*webhookclient.APIClient, error) {
	profile := config.GetCurrentProfile(cmd, cfg)

	organizationID, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
	if err != nil {
		return nil, err
	}

	stack, err := cmdbuilder.ResolveStack(cmd, cfg, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := config.GetHttpClient(cmd)

	token, err := profile.GetStackToken(cmd.Context(), httpClient, stack)
	if err != nil {
		return nil, err
	}

	apiConfig := webhookclient.NewConfiguration()
	apiConfig.Servers = webhookclient.ServerConfigurations{{
		URL: profile.ApiUrl(stack, "webhooks").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return webhookclient.NewAPIClient(apiConfig), nil
}

func NewCreateCommand() *cobra.Command {
	const (
		secretFlag = "secret"
	)
	return cmdbuilder.NewCommand("create",
		cmdbuilder.WithShortDescription("Create a new webhook"),
		cmdbuilder.WithAliases("cr"),
		cmdbuilder.WithArgs(cobra.MinimumNArgs(2)),
		cmdbuilder.WithStringFlag(secretFlag, "", "Webhook secret"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}
			webhookClient, err := newWebhookClient(cmd, cfg)
			if err != nil {
				return err
			}

			if _, err := url.Parse(args[0]); err != nil {
				return err
			}

			secret := cmdutils.GetString(cmd, secretFlag)

			response, _, err := webhookClient.ConfigsApi.InsertOneConfig(cmd.Context()).ConfigUser(webhookclient.ConfigUser{
				Endpoint:   &args[0],
				EventTypes: args[1:],
				Secret:     &secret,
			}).Execute()
			if err != nil {
				return err
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Config created with ID: %s", strings.TrimSuffix(response, "\n"))
			return nil
		}),
	)
}
