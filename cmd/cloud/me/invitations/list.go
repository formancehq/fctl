package invitations

import (
	"fmt"
	"time"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
	store            *ListStore
	statusFlag       string
	organizationFlag string
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store:            NewDefaultListStore(),
		statusFlag:       "status",
		organizationFlag: "organization",
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List invitations"),
		fctl.WithStringFlag(c.statusFlag, "", "Filter invitations by status"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithStringFlag(c.organizationFlag, "", "Filter invitations by organization"),
		fctl.WithController[*ListStore](c),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	status := fctl.GetString(cmd, c.statusFlag)
	organization := fctl.GetString(cmd, c.organizationFlag)

	request := operations.ListInvitationsRequest{}
	if status != "" {
		request.Status = &status
	}
	if organization != "" {
		request.Organization = &organization
	}

	response, err := apiClient.ListInvitations(cmd.Context(), request)
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

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	tableData := fctl.Map(c.store.Invitations, func(i Invitations) []string {
		return []string{
			i.Id,
			i.UserEmail,
			i.Status,
			i.CreationDate.Format(time.RFC3339),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "Email", "Status", "CreationDate"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
