package cmd

import (
	"fmt"

	"github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newSandboxListCommand() *cobra.Command {
	return newMembershipCommand("list",
		withShortDescription("list sandboxes"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			organization, err := fctl.FindOrganizationId(cmd.Context())
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient := fctl.NewMembershipClientFromContext(cmd.Context())

			rsp, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
			if err != nil {
				return errors.Wrap(err, "listing sandboxs")
			}

			if len(rsp.Data) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No sandboxs found.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Stacks: ")
			for _, s := range rsp.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", s.Id, s.Name)
			}
			return nil
		}),
	)
}
