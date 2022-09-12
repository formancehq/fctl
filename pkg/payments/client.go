package payments

import (
	"fmt"

	fctl "github.com/numary/fctl/pkg"
	"github.com/numary/payments/client"
)

func NewClient(profile *fctl.Profile, debug bool, organization, stack string) *client.APIClient {
	configuration := client.NewConfiguration()
	configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", profile.Token))
	configuration.Servers[0].URL = fctl.MustApiUrl(profile, organization, stack, "ledger").String()
	configuration.Debug = debug
	return client.NewAPIClient(configuration)
}
