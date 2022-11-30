package stack

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	return internal.NewMembershipCommand("list",
		internal.WithAliases("ls", "l"),
		internal.WithShortDescription("List sandboxes"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			profile := internal.GetCurrentProfile(cmd, cfg)

			organization, err := internal.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := internal.NewMembershipClient(cmd, cfg)
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

			tableData := internal.Map(rsp.Data, func(stack membershipclient.Stack) []string {
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
			tableData = internal.Prepend(tableData, []string{"ID", "Name", "Region", "Dashboard"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
