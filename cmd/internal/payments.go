package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/numary/payments/client"
)

func NewPaymentsClientFromContext(ctx context.Context, profile *Profile, httpClient *http.Client, organizationID, stackID string) (*client.APIClient, error) {
	token, err := profile.GetToken(ctx, httpClient)
	if err != nil {
		return nil, err
	}

	config := client.NewConfiguration()
	config.Servers = client.ServerConfigurations{{
		URL: profile.ApiUrl(organizationID, stackID, "payments").String(),
	}}
	config.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	config.HTTPClient = httpClient

	return client.NewAPIClient(config), nil
}
