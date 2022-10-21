package fctl

import (
	"context"
	"fmt"

	"github.com/numary/payments/client"
)

func NewPaymentsClientFromContext(ctx context.Context) (*client.APIClient, error) {
	token, err := CurrentProfileFromContext(ctx).GetStackToken(ctx)
	if err != nil {
		return nil, err
	}

	organization, err := FindOrganizationId(ctx)
	if err != nil {
		return nil, err
	}

	stack, err := FindStackId(ctx, organization)
	if err != nil {
		return nil, err
	}

	profile := CurrentProfileFromContext(ctx)
	config := client.NewConfiguration()
	config.Servers = client.ServerConfigurations{{
		URL: MustApiUrl(*profile, organization, stack, "payments").String(),
	}}
	config.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	config.HTTPClient = NewHTTPClientFromContext(ctx)

	return client.NewAPIClient(config), nil
}
