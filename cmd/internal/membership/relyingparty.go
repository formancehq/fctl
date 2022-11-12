package membership

import (
	"context"

	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/debugutils"
	"github.com/zitadel/oidc/pkg/client/rp"
)

func GetRelyingParty(ctx context.Context, profile *config.Profile) (rp.RelyingParty, error) {
	return rp.NewRelyingPartyOIDC(profile.GetMembershipURI(), config.AuthClient, "",
		"", []string{"openid", "email", "offline_access", "supertoken"}, rp.WithHTTPClient(debugutils.GetHttpClient(ctx)))
}
