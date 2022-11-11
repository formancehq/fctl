package users

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return cmdbuilder.NewCommand("show",
		cmdbuilder.WithAliases("s"),
		cmdbuilder.WithShortDescription("Show user"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			client, err := internal.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			readUserResponse, _, err := client.DefaultApi.ReadUser(cmd.Context(), args[0]).Execute()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "-> User ID: %s\r\n", *readUserResponse.Data.Id)
			fmt.Fprintf(cmd.OutOrStdout(), "Membership user ID: %s\r\n", *readUserResponse.Data.Subject)
			fmt.Fprintf(cmd.OutOrStdout(), "Email: %s\r\n", *readUserResponse.Data.Email)
			return nil
		}),
	)
}
