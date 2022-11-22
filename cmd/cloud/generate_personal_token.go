package cloud

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewGeneratePersonalTokenCommand() *cobra.Command {
	return cmdbuilder.NewCommand("generate-personal-token",
		cmdbuilder.WithDescription("Generate a personal bearer token"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}
			profile := config.GetCurrentProfile(cmd, cfg)

			organizationID, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := cmdbuilder.ResolveStack(cmd, cfg, organizationID)
			if err != nil {
				return err
			}

			token, err := profile.GetStackToken(cmd.Context(), config.GetHttpClient(cmd), stack)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), token)
			return nil
		}),
	)
}
