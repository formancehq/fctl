package invitations

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Invitations struct {
	Id           string    `json:"id"`
	UserEmail    string    `json:"userEmail"`
	Status       string    `json:"status"`
	CreationDate time.Time `json:"creationDate"`
}

type ListStore struct {
	Invitations []Invitations `json:"invitations"`
}
type ListController struct {
	store      *ListStore
	statusFlag string
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store:      NewDefaultListStore(),
		statusFlag: "status",
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithShortDescription("List invitations"),
		fctl.WithAliases("s"),
		fctl.WithStringFlag(c.statusFlag, "", "Filter invitations by status"),
		fctl.WithController[*ListStore](c),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	status := fctl.GetString(cmd, c.statusFlag)
	request := operations.ListOrganizationInvitationsRequest{
		OrganizationID: organizationID,
	}
	if status != "" {
		request.Status = &status
	}

	response, err := apiClient.ListOrganizationInvitations(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ListInvitationsResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Invitations = fctl.Map(response.ListInvitationsResponse.GetData(), func(i components.Invitation) Invitations {
		return Invitations{
			Id:           i.GetID(),
			UserEmail:    i.GetUserEmail(),
			Status:       string(i.GetStatus()),
			CreationDate: i.GetCreationDate(),
		}
	})

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	tableData := fctl.Map(c.store.Invitations, func(i Invitations) []string {
		return []string{
			i.Id,
			i.UserEmail,
			i.Status,
			i.CreationDate.Format(time.RFC3339),
		}
	})

	tableData = fctl.Prepend(tableData, []string{"ID", "Email", "Status", "Creation date", "Org claim"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
