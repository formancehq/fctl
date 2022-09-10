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
		currentProfile.AccessToken, err = membershiplogin.LogIn(currentProfile.MembershipURI)
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
