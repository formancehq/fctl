package membership

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/numary/membership-api/client"
)

func NewClient(profile fctl.Profile, debug bool) *client.APIClient {
	configuration := client.NewConfiguration()
	if profile.Tokens != nil {
		configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", profile.Tokens.AccessToken))
	}
	configuration.Servers[0].URL = profile.MembershipURI
	configuration.Debug = debug
	return client.NewAPIClient(configuration)
}
