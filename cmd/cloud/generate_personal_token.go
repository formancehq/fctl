package cloud

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewGeneratePersonalTokenCommand() *cobra.Command {
	return internal.NewCommand("generate-personal-token",
		internal.WithDescription("Generate a personal bearer token"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}
			profile := internal.GetCurrentProfile(cmd, cfg)

			organizationID, err := internal.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := internal.ResolveStack(cmd, cfg, organizationID)
			if err != nil {
				return err
			}

			token, err := profile.GetStackToken(cmd.Context(), internal.GetHttpClient(cmd), stack)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), token)
			return nil
		}),
	)
}
