package cmd

import (
	"fmt"

	"github.com/numary/membership-api/pkg/membershiplogin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/pkg/client/rp"
)

const authClient = "fctl"

var loginCommand = &cobra.Command{
	Use: "login",
	RunE: func(cmd *cobra.Command, args []string) (err error) {

		relyingParty, err := rp.NewRelyingPartyOIDC(currentProfile.MembershipURI, authClient, "",
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
		currentProfile.Token = ret.Token

		if err := configManager.UpdateConfig(config); err != nil {
			return errors.Wrap(err, "updating config")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Logged!")
		return nil
	},
}

func init() {
	rootCommand.AddCommand(loginCommand)
}
