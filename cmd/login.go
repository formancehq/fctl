package cmd

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/numary/membership-api/pkg/membershiplogin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/pkg/client/rp"
)

const (
	membershipUriFlag  = "membership-uri"
	baseServiceUriFlag = "service-uri"

	// TODO: Make configurable at build
	authClient = "fctl"
)

func newLoginCommand() *cobra.Command {
	return newCommand("login",
		withStringFlag(membershipUriFlag, fctl.DefaultMemberShipUri, "service url"),
		withStringFlag(baseServiceUriFlag, fctl.DefaultBaseUri, "service url"),
		withHiddenFlag(membershipUriFlag),
		withHiddenFlag(baseServiceUriFlag),
		withRunE(func(cmd *cobra.Command, args []string) error {
			relyingParty, err := rp.NewRelyingPartyOIDC(fctl.CurrentProfileFromContext(cmd.Context()).MembershipURI, authClient, "",
				"", []string{"openid", "email", "offline_access"})
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
			fctl.CurrentProfileFromContext(cmd.Context()).Token = ret.Token
			fctl.ConfigFromContext(cmd.Context()).CurrentProfile = fctl.CurrentProfileNameFromContext(cmd.Context())

			if err := fctl.ConfigManagerFromContext(cmd.Context()).
				UpdateConfig(fctl.ConfigFromContext(cmd.Context())); err != nil {
				return errors.Wrap(err, "updating config")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged!")
			return nil
		}),
	)
}
