package membership

import (
	"fmt"
	"net/http"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/numary/membership-api/client"
)

func NewClient(profile fctl.Profile, httpClient *http.Client, debug bool) *client.APIClient {
	configuration := client.NewConfiguration()
	if profile.Token != nil {
		configuration.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", profile.Token.AccessToken))
	}
	configuration.HTTPClient = httpClient

	configuration.Servers[0].URL = profile.MembershipURI
	//configuration.Debug = debug
	return client.NewAPIClient(configuration)
}
