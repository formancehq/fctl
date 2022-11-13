package invitations

import (
	"time"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	const (
		statusFlag       = "status"
		organizationFlag = "organization"
	)
	return cmdbuilder.NewCommand("list",
		cmdbuilder.WithAliases("ls", "l"),
		cmdbuilder.WithShortDescription("List invitations"),
		cmdbuilder.WithStringFlag(statusFlag, "", "Filter invitations by status"),
		cmdbuilder.WithStringFlag(organizationFlag, "", "Filter invitations by organization"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}
			client, err := config.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			listInvitationsResponse, _, err := client.DefaultApi.
				ListInvitations(cmd.Context()).
				Status(cmdutils.GetString(cmd, statusFlag)).
				Organization(cmdutils.GetString(cmd, organizationFlag)).
				Execute()
			if err != nil {
				return err
			}

			tableData := collections.Map(listInvitationsResponse.Data, func(i membershipclient.Invitation) []string {
				return []string{
					i.Id,
					i.UserEmail,
					i.Status,
					i.CreationDate.Format(time.RFC3339),
				}
			})
			tableData = collections.Prepend(tableData, []string{"ID", "Email", "Status", "CreationDate"})
			return pterm.DefaultTable.
				WithHasHeader().
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
