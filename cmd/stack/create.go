package stack

import (
	"fmt"

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
		cmdbuilder.WithShortDescription("create a new sandbox"),
		cmdbuilder.WithAliases("c"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get()
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

			sandbox, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
				Name: args[0],
			}).Execute()
			if err != nil {
				return errors.Wrap(err, "creating sandbox")
			}

			profile, err := config.GetCurrentProfile(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Stack created with ID: %s\r\n", sandbox.Data.Id)

			return internal.PrintStackInformation(cmd.OutOrStdout(), profile, sandbox.Data)
		}),
	)
}
