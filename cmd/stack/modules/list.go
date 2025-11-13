package modules

import (
	"fmt"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/time"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ListStore struct {
	Modules []components.Module `json:"modules"`
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
	return fctl.NewMembershipCommand("list --stack=<stack-id>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("List modules in a stack"),
		fctl.WithAliases("ls"),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewListController()),
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

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	request := operations.ListModulesRequest{
		OrganizationID: organizationID,
		StackID:        stackID,
	}

	response, err := apiClient.ListModules(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ListModulesResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Modules = response.ListModulesResponse.GetData()

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	header := []string{"Name", "State", "Cluster status", "Last state update", "Last cluster state update"}

	tableData := fctl.Map(c.store.Modules, func(module components.Module) []string {
		return []string{
			module.GetName(),
			string(module.GetState()),
			string(module.GetStatus()),
			time.Time{Time: module.GetLastStateUpdate()}.String(),
			time.Time{Time: module.GetLastStatusUpdate()}.String(),
		}
	})

	tableData = fctl.Prepend(tableData, header)

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithHasHeader().
		WithData(tableData).
		Render()
}
