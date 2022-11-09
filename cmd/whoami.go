package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWhoamiCommand() *cobra.Command {
	return newCommand("whoami",
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

			userInfo, err := profile.GetUserInfo(cmd.Context(), relyingParty)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Subject: %s\r\n", userInfo.GetSubject())
			fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\r\n", userInfo.GetEmail())

			return nil
		}),
	)
}
