package profiles

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newProfilesShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			config, err := config.Get()
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
