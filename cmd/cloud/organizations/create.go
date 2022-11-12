package organizations

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	return cmdbuilder.NewCommand("create",
		cmdbuilder.WithAliases("cr", "c"),
		cmdbuilder.WithShortDescription("Create organization"),
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

			response, _, err := apiClient.DefaultApi.
				CreateOrganization(cmd.Context()).
				Body(membershipclient.OrganizationData{
					Name: args[0],
				}).Execute()
			if err != nil {
				return err
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Organization '%s' created with ID: %s", args[0], response.Data.Id)

			return nil
		}),
	)
}
