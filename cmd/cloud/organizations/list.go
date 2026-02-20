package organizations

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"
	"github.com/formancehq/go-libs/v3/pointer"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type OrgRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	OwnerID    string `json:"ownerId"`
	OwnerEmail string `json:"ownerEmail"`
	Domain     string `json:"domain"`
	IsMine     string `json:"isMine"`
}

type ListStore struct {
	Organizations []*OrgRow `json:"organizations"`
}
type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultListStore(),
	}
}

func NewListCommand() *cobra.Command {
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List organizations"),
		fctl.WithBoolFlag("expand", true, "Expand the organization"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewListController()),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	expand := fctl.GetBool(cmd, "expand")

	request := operations.ListOrganizationsRequest{
		Expand: pointer.For(expand),
	}

	response, err := apiClient.ListOrganizations(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ListOrganizationExpandedResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	claims, err := profile.GetClaims()
	if err != nil {
		return nil, err
	}

	c.store.Organizations = fctl.Map(response.ListOrganizationExpandedResponse.GetData(), func(o components.OrganizationExpanded) *OrgRow {
		isMine := fctl.BoolToString(o.GetOwnerID() == claims.Subject)
		return &OrgRow{
			ID:      o.GetID(),
			Name:    o.GetName(),
			OwnerID: o.GetOwnerID(),
			OwnerEmail: func() string {
				if owner := o.GetOwner(); owner != nil {
					return owner.GetEmail()
				}
				return ""
			}(),
			Domain: func() string {
				if domain := o.GetDomain(); domain != nil {
					return *domain
				}
				return ""
			}(),
			IsMine: isMine,
		}
	})

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	OrgMap := fctl.Map(c.store.Organizations, func(o *OrgRow) []string {
		return []string{o.ID, o.Name, o.OwnerID, o.OwnerEmail, o.Domain, o.IsMine}
	})

	tableData := fctl.Prepend(OrgMap, []string{"ID", "Name", "Owner ID", "Owner email", "Domain", "Is mine?"})

	return pterm.DefaultTable.WithHasHeader().WithWriter(cmd.OutOrStdout()).WithData(tableData).Render()
}
