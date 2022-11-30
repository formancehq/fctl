package invitations

import (
	"time"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	const (
		statusFlag = "status"
	)
	return internal.NewCommand("list",
		internal.WithAliases("ls", "l"),
		internal.WithArgs(cobra.ExactArgs(0)),
		internal.WithShortDescription("List invitations"),
		internal.WithAliases("s"),
		internal.WithStringFlag(statusFlag, "", "Filter invitations by status"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			apiClient, err := internal.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			organizationID, err := internal.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			listInvitationsResponse, _, err := apiClient.DefaultApi.
				ListOrganizationInvitations(cmd.Context(), organizationID).
				Status(internal.GetString(cmd, statusFlag)).
				Execute()
			if err != nil {
				return err
			}

			tableData := internal.Map(listInvitationsResponse.Data, func(i membershipclient.Invitation) []string {
				return []string{
					i.Id,
					i.UserEmail,
					i.Status,
					i.CreationDate.Format(time.RFC3339),
				}
			})
			tableData = internal.Prepend(tableData, []string{"ID", "Email", "Status", "Creation date"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
