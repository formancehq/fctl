package payments

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/numary/payments/client"
)

func NewClient(profile *fctl.Profile, debug bool, organization, stack string) *client.APIClient {
	configuration := client.NewConfiguration()
	configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", profile.Tokens.AccessToken))
	configuration.Servers[0].URL = fctl.MustApiUrl(profile, organization, stack, "payments").String()
	configuration.Debug = debug
	return client.NewAPIClient(configuration)
}
