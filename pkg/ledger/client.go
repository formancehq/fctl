package ledger

import (
	fctl "github.com/formancehq/fctl/pkg"
	ledgerclient "github.com/numary/ledger/client"
)

func NewClient(profile *fctl.Profile, debug bool, organization, stack string) *ledgerclient.APIClient {
	config := ledgerclient.NewConfiguration()
	config.Servers = ledgerclient.ServerConfigurations{{
		URL: fctl.MustApiUrl(profile, organization, stack, "ledger").String(),
	}}
	config.Debug = debug
	return ledgerclient.NewAPIClient(config)
}
