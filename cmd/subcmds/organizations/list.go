package organizations

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	internal2 "github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/spf13/cobra"
)

func NewOrganizationsListCommand() *cobra.Command {
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithShortDescription("list organizations"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			config, err := internal2.GetConfig()
			if err != nil {
				return err
			}

			apiClient, err := membership.NewMembershipClient(cmd, config)
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
