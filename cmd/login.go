package cmd

import (
	"fmt"

	"github.com/numary/membership-api/pkg/membershiplogin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/pkg/client/rp"
)

var loginCommand = &cobra.Command{
	Use: "login",
	RunE: func(cmd *cobra.Command, args []string) (err error) {

		relyingParty, err := rp.NewRelyingPartyOIDC(currentProfile.MembershipURI, "fctl", "",
			"", []string{"openid", "email", "offline_access"})
		if err != nil {
			return err
		}

		currentProfile.Tokens, err = membershiplogin.LogIn(cmd.Context(), membershiplogin.DialogFn(func(uri, code string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Please enter the following code on your browser:", code)
			fmt.Fprintln(cmd.OutOrStdout(), "Link:", uri)
		}), relyingParty)
		if err != nil {
			return err
		}
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
