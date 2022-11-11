package membership

import (
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/internal/debugutil"
	"github.com/zitadel/oidc/pkg/client/rp"
)

func GetRelyingParty(profile *config.Profile) (rp.RelyingParty, error) {
	return rp.NewRelyingPartyOIDC(profile.GetMembershipURI(), config.AuthClient, "",
		"", []string{"openid", "email", "offline_access", "supertoken"}, rp.WithHTTPClient(debugutil.GetHttpClient()))
}
