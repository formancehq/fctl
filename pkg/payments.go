package fctl

import (
	"context"
	"fmt"

	"github.com/numary/payments/client"
)

func NewPaymentsClientFromContext(ctx context.Context) (*client.APIClient, error) {
	organization, err := FindOrganizationId(ctx)
	if err != nil {
		return nil, err
	}

	stackId, err := FindStackId(ctx, organization)
	if err != nil {
		return nil, err
	}

	token, err := GetToken(ctx, *CurrentProfileFromContext(ctx), organization, stackId)
	if err != nil {
		return nil, err
	}

	profile := CurrentProfileFromContext(ctx)
	config := client.NewConfiguration()
	config.Servers = client.ServerConfigurations{{
		URL: MustApiUrl(*profile, organization, StackFromContext(ctx), "payments").String(),
	}}
	config.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	config.HTTPClient = HttpClientFromContext(ctx)

	return client.NewAPIClient(config), nil
}
