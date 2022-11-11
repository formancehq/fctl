package invitations

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
)

func NewDeclineCommand() *cobra.Command {
	return cmdbuilder.NewCommand("decline",
		cmdbuilder.WithAliases("dec", "d"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			client, err := membership.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			_, err = client.DefaultApi.DeclineInvitation(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			fmt.Println("Invitation declined!")
			return nil
		}),
	)
}
