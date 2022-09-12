package membership

import (
	"fmt"

	fctl "github.com/numary/fctl/pkg"
	"github.com/numary/membership-api/client"
)

func NewClient(profile fctl.Profile, debug bool) *client.APIClient {
	configuration := client.NewConfiguration()
	if profile.Token != nil {
		configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", profile.Token.AccessToken))
	}
	configuration.Servers[0].URL = profile.MembershipURI
	configuration.Debug = debug
	return client.NewAPIClient(configuration)
}
