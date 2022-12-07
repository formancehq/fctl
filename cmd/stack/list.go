package stack

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return fctl.NewMembershipCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List sandboxes"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			profile := fctl.GetCurrentProfile(cmd, cfg)

			organization, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := fctl.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			rsp, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
			if err != nil {
				return errors.Wrap(err, "listing sandboxs")
			}

			if len(rsp.Data) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No sandboxs found.")
				return nil
			}

			tableData := fctl.Map(rsp.Data, func(stack membershipclient.Stack) []string {
				return []string{
					stack.Id,
					stack.Name,
					func() string {
						if stack.Region == nil {
							return ""
						}
						return *stack.Region
					}(),
					profile.ServicesBaseUrl(&stack).String(),
				}
			})
			tableData = fctl.Prepend(tableData, []string{"ID", "Name", "Region", "Dashboard"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
