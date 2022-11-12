package config

import (
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/pkg/client/rp"
)

func GetAuthRelyingParty(cmd *cobra.Command, profile *Profile) (rp.RelyingParty, error) {
	return rp.NewRelyingPartyOIDC(profile.GetMembershipURI(), AuthClient, "",
		"", []string{"openid", "email", "offline_access", "supertoken"}, rp.WithHTTPClient(GetHttpClient(cmd)))
}
