package cmd

import (
	"fmt"

	fctl "github.com/formancehq/fctl/cmd/internal"
	membershipclient "github.com/numary/membership-api/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newSandboxCreateCommand() *cobra.Command {
	return newCommand("create",
		withShortDescription("create a new sandbox"),
		withArgs(cobra.ExactArgs(1)),
		withPersistentStringFlag(organizationFlag, "", "Specific organization to target"),
		withRunE(func(cmd *cobra.Command, args []string) error {

			organization, err := resolveOrganizationID(cmd)
			if err != nil {
				return err
			}

			apiClient, err := newMembershipClient(cmd)
			if err != nil {
				return err
			}

			sandbox, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
				Name: args[0],
			}).Execute()
			if err != nil {
				return errors.Wrap(err, "creating sandbox")
			}

			profile, err := getCurrentProfile()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Stack created with ID: %s\r\n", sandbox.Data.Id)

			return fctl.PrintStackInformation(cmd.OutOrStdout(), profile, sandbox.Data)
		}),
	)
}
