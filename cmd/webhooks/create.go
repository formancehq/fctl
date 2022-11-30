package webhooks

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/formancehq/fctl/cmd/internal"
	webhookclient "github.com/formancehq/webhooks/client"
	"github.com/spf13/cobra"
)

func NewStackClient(cmd *cobra.Command, cfg *internal.Config) (*webhookclient.APIClient, error) {
	profile := internal.GetCurrentProfile(cmd, cfg)

	organizationID, err := internal.ResolveOrganizationID(cmd, cfg)
	if err != nil {
		return nil, err
	}

	stack, err := internal.ResolveStack(cmd, cfg, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := internal.GetHttpClient(cmd)

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
	return internal.NewCommand("create",
		internal.WithShortDescription("Create a new webhook"),
		internal.WithAliases("cr"),
		internal.WithArgs(cobra.MinimumNArgs(2)),
		internal.WithStringFlag(secretFlag, "", "Webhook secret"),
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

			secret := internal.GetString(cmd, secretFlag)

			response, _, err := webhookClient.ConfigsApi.InsertOneConfig(cmd.Context()).ConfigUser(webhookclient.ConfigUser{
				Endpoint:   &args[0],
				EventTypes: args[1:],
				Secret:     &secret,
			}).Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(), "Config created with ID: %s", strings.TrimSuffix(response, "\n"))
			return nil
		}),
	)
}
