package organizations

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("ls", "l"),
		cmdbuilder.WithShortDescription("List organizations"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			apiClient, err := membership.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			organizations, _, err := apiClient.DefaultApi.ListOrganizations(cmd.Context()).Execute()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Organizations: ")
			for _, o := range organizations.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "-> Organization: %s\r\n", o.Id)
				PrintOrganization(cmd.OutOrStdout(), o)
			}
			return nil
		}),
	)
}
