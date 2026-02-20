package policies

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/time"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Policies []components.Policy `json:"policies"`
}

type ListController struct {
	store *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{
		Policies: []components.Policy{},
	}
}

func NewListController() *ListController {
	return &ListController{
		store: NewDefaultListStore(),
	}
}

func NewListCommand() *cobra.Command {
	return fctl.NewCommand(`list`,
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List organization policies"),
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

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	request := operations.ListPoliciesRequest{
		OrganizationID: organizationID,
	}

	response, err := apiClient.ListPolicies(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ListPoliciesResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Policies = response.ListPoliciesResponse.GetData()

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	if len(c.store.Policies) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No policies found.")
		return nil
	}

	header := []string{"ID", "Name", "Description", "Protected", "Created At", "Updated At"}
	tableData := fctl.Map(c.store.Policies, func(policy components.Policy) []string {
		return []string{
			fmt.Sprintf("%d", policy.GetID()),
			policy.GetName(),
			func() string {
				if desc := policy.GetDescription(); desc != nil {
					return *desc
				}
				return ""
			}(),
			fctl.BoolToString(policy.GetProtected()),
			time.Time{Time: policy.GetCreatedAt()}.String(),
			time.Time{Time: policy.GetUpdatedAt()}.String(),
		}
	})

	tableData = fctl.Prepend(tableData, header)

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
