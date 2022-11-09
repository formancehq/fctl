package cmd

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/numary/membership-api/pkg/membershiplogin"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/pkg/client/rp"
)

const (
	membershipUriFlag  = "membership-uri"
	baseServiceUriFlag = "service-uri"
)

func getRelyingParty(profile *internal.Profile) (rp.RelyingParty, error) {
	return rp.NewRelyingPartyOIDC(profile.GetMembershipURI(), internal.AuthClient, "",
		"", []string{"openid", "email", "offline_access"}, rp.WithHTTPClient(getHttpClient()))

}

func newLoginCommand() *cobra.Command {
	return newCommand("login",
		withStringFlag(membershipUriFlag, internal.DefaultMemberShipUri, "service url"),
		withStringFlag(baseServiceUriFlag, internal.DefaultBaseUri, "service url"),
		withHiddenFlag(membershipUriFlag),
		withHiddenFlag(baseServiceUriFlag),
		withRunE(func(cmd *cobra.Command, args []string) error {

			config, err := getConfig()
			if err != nil {
				return err
			}

			profile, err := getCurrentProfile(config)
			if err != nil {
				return err
			}

			relyingParty, err := getRelyingParty(profile)
			if err != nil {
				return err
			}

			ret, err := membershiplogin.LogIn(cmd.Context(), membershiplogin.DialogFn(func(uri, code string) {
				fmt.Fprintln(cmd.OutOrStdout(), "Please enter the following code on your browser:", code)
				fmt.Fprintln(cmd.OutOrStdout(), "Link:", uri)
			}), relyingParty)
			if err != nil {
				return err
			}

			profile.UpdateToken(ret.Token)

			currentProfileName, err := getCurrentProfileName()
			if err != nil {
				return err
			}

			config.SetCurrentProfile(currentProfileName, profile)

			fmt.Fprintln(cmd.OutOrStdout(), "Logged!")
			return config.Persist()
		}),
	)
}
