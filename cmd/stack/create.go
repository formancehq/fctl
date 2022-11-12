package stack

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	return cmdbuilder.NewMembershipCommand("create",
		cmdbuilder.WithShortDescription("Create a new sandbox"),
		cmdbuilder.WithAliases("c", "cr"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			organization, err := cmdbuilder.ResolveOrganizationID(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			apiClient, err := membership.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			stack, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
				Name: args[0],
			}).Execute()
			if err != nil {
				return errors.Wrap(err, "creating sandbox")
			}

			profile := config.GetCurrentProfile(cmd.Context(), cfg)

			cmdbuilder.Highlightln(cmd.OutOrStdout(), "Your dashboard will be reachable on: %s",
				profile.ServicesBaseUrl(stack.Data.OrganizationId, stack.Data.Id).String())
			cmdbuilder.Highlightln(cmd.OutOrStdout(), "You can access your sandbox apis using following urls :")

			return internal.PrintStackInformation(cmd.OutOrStdout(), profile, stack.Data)
		}),
	)
}
