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
		statusFlag       = "status"
		organizationFlag = "organization"
	)
	return internal.NewCommand("list",
		internal.WithAliases("ls", "l"),
		internal.WithShortDescription("List invitations"),
		internal.WithStringFlag(statusFlag, "", "Filter invitations by status"),
		internal.WithStringFlag(organizationFlag, "", "Filter invitations by organization"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}
			client, err := internal.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			listInvitationsResponse, _, err := client.DefaultApi.
				ListInvitations(cmd.Context()).
				Status(internal.GetString(cmd, statusFlag)).
				Organization(internal.GetString(cmd, organizationFlag)).
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
			tableData = internal.Prepend(tableData, []string{"ID", "Email", "Status", "CreationDate"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
