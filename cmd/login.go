package cmd

import (
	"fmt"

	"github.com/numary/membership-api/pkg/membershiplogin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var loginCommand = &cobra.Command{
	Use: "login",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		currentProfile.Token, err = membershiplogin.LogIn(cmd.Context(), membershiplogin.DialogFn(func(uri, code string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Please enter the following code on your browser:", code)
		}), currentProfile.MembershipURI)
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
