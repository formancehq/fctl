package organizations

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	return internal.NewCommand("create",
		internal.WithAliases("cr", "c"),
		internal.WithShortDescription("Create organization"),
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			apiClient, err := internal.NewMembershipClient(cmd, cfg)
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

			internal.Success(cmd.OutOrStdout(), "Organization '%s' created with ID: %s", args[0], response.Data.Id)

			return nil
		}),
	)
}
