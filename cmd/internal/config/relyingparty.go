package config

import (
	"context"

	"github.com/zitadel/oidc/pkg/client/rp"
)

func GetAuthRelyingParty(ctx context.Context, profile *Profile) (rp.RelyingParty, error) {
	return rp.NewRelyingPartyOIDC(profile.GetMembershipURI(), AuthClient, "",
		"", []string{"openid", "email", "offline_access", "supertoken"}, rp.WithHTTPClient(GetHttpClient(ctx)))
}
