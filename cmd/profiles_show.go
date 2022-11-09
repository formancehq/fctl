package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesShowCommand() *cobra.Command {
	return newCommand("show",
		withArgs(cobra.ExactArgs(1)),
		withRunE(func(cmd *cobra.Command, args []string) error {

			config, err := getConfig()
			if err != nil {
				return err
			}
			p := config.GetProfile(args[0])
			if p == nil {
				return errors.New("not found")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Domain:", p.GetBaseServiceURI())
			fmt.Fprintln(cmd.OutOrStdout(), "Membership:", p.GetMembershipURI())
			return nil
		}),
	)
}
