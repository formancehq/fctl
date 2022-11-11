package users

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("ls", "l"),
		cmdbuilder.WithShortDescription("List users"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			client, err := internal.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			listUsersResponse, _, err := client.DefaultApi.ListUsers(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			for _, user := range listUsersResponse.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "-> User ID: %s\r\n", *user.Id)
				fmt.Fprintf(cmd.OutOrStdout(), "Membership user ID: %s\r\n", *user.Subject)
				fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\r\n", *user.Email)
			}
			return nil
		}),
	)
}
