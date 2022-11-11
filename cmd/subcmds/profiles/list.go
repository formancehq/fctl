package profiles

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/spf13/cobra"
)

func newProfilesListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("l"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get()
			if err != nil {
				return err
			}

			currentProfileName, err := config.GetCurrentProfileName()
			if err != nil {
				return err
			}

			for p := range cfg.GetProfiles() {
				fmt.Fprint(cmd.OutOrStdout(), "- ", p)
				if currentProfileName == p {
					fmt.Fprint(cmd.OutOrStdout(), " *")
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		}))
}
