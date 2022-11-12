package internal

import (
	"context"
	"fmt"

	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
)

func NewAuthClient(ctx context.Context, cfg *config.Config) (*authclient.APIClient, error) {
	profile := config.GetCurrentProfile(ctx, cfg)

	organizationID, err := cmdbuilder.ResolveOrganizationID(ctx, cfg)
	if err != nil {
		return nil, err
	}

	stackID, err := cmdbuilder.ResolveStackID(ctx, cfg, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := config.GetHttpClient(ctx)

	token, err := profile.GetStackToken(ctx, httpClient, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	apiConfig := authclient.NewConfiguration()
	apiConfig.Servers = authclient.ServerConfigurations{{
		URL: profile.ApiUrl(organizationID, stackID, "auth").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return authclient.NewAPIClient(apiConfig), nil
}
