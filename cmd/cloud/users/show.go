package users

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show",
		cmdbuilder.WithAliases("s"),
		cmdbuilder.WithShortDescription("show user by id"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			apiClient, err := membership.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			organizationID, err := cmdbuilder.ResolveOrganizationID(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			userResponse, _, err := apiClient.DefaultApi.ReadUser(cmd.Context(), organizationID, args[0]).Execute()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "User: ")
			fmt.Fprintf(cmd.OutOrStdout(), "-> User: %s\r\n", userResponse.Data.Id)
			fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\r\n", userResponse.Data.Email)
			return nil
		}),
	)
}
