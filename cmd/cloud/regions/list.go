package regions

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type ListStore struct {
	Regions []components.AnyRegion `json:"regions"`
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
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List regions"),
		fctl.WithController[*ListStore](NewListController()),
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

	request := operations.ListRegionsRequest{
		OrganizationID: organizationID,
	}

	response, err := apiClient.ListRegions(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ListRegionsResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Regions = response.ListRegionsResponse.GetData()

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	tableData := fctl.Map(c.store.Regions, func(i components.AnyRegion) []string {
		return []string{
			i.GetID(),
			i.GetName(),
			i.GetBaseURL(),
			fctl.BoolToString(i.GetPublic()),
			fctl.BoolToString(i.GetActive()),
			func() string {
				if ping := i.GetLastPing(); ping != nil {
					return ping.Format(time.RFC3339)
				}
				return ""
			}(),
			func() string {
				if creator := i.GetCreator(); creator != nil {
					return creator.GetEmail()
				}
				return "Formance Cloud"
			}(),
			func() string {
				if version := i.GetVersion(); version != nil {
					return *version
				}
				return ""
			}(),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "Name", "Base url", "Public", "Active", "Last ping", "Owner", "Version"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
