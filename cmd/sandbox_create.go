package cmd

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
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

			baseUrl, err := fctl.ServicesBaseUrl(*profile, sandbox.Data.OrganizationId, sandbox.Data.Id)
			if err != nil {
				return err
			}
			baseUrlStr := baseUrl.String()

			fmt.Fprintf(cmd.OutOrStdout(), "Stack created with ID: %s\r\n", sandbox.Data.Id)
			fmt.Fprintf(cmd.OutOrStdout(), "Your dashboard will be reachable on: %s\r\n", baseUrlStr)
			fmt.Fprintln(cmd.OutOrStdout(), "You can access your sandbox apis using following urls :")
			fmt.Fprintf(cmd.OutOrStdout(), "Ledger: %s/api/ledger\r\n", baseUrlStr)
			fmt.Fprintf(cmd.OutOrStdout(), "Payments: %s/api/payments\n", baseUrlStr)
			fmt.Fprintf(cmd.OutOrStdout(), "Search: %s/api/search\n", baseUrlStr)
			fmt.Fprintf(cmd.OutOrStdout(), "Auth: %s/api/auth\n", baseUrlStr)

			return nil
		}),
	)
}
