package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var showProfileCommand = &cobra.Command{
	Use:  "show",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := config.GetProfile(args[0])
		if p == nil {
			return errors.New("not found")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Domain:", p.BaseServiceURI)
		fmt.Fprintln(cmd.OutOrStdout(), "Membership:", p.MembershipURI)
		return nil
	},
}

func init() {
	profilesCommand.AddCommand(showProfileCommand)
}
