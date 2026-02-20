package policies

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/time"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	Policy *components.Policy `json:"policy"`
}

type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewDefaultShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand(`show <policy-id>`,
		fctl.WithAliases("s", "sh"),
		fctl.WithShortDescription("Show policy details"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewShowController()),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	policyID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid policy ID: %w", err)
	}

	request := operations.ReadPolicyRequest{
		OrganizationID: organizationID,
		PolicyID:       policyID,
	}

	response, err := apiClient.ReadPolicy(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ReadPolicyResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.Policy = response.ReadPolicyResponse.GetData()

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	if c.store.Policy == nil {
		return fmt.Errorf("no policy data")
	}

	policy := c.store.Policy

	data := [][]string{
		{"ID", fmt.Sprintf("%d", policy.GetID())},
		{"Name", policy.GetName()},
		{"Description", func() string {
			if desc := policy.GetDescription(); desc != nil {
				return *desc
			}
			return ""
		}()},
		{"Protected", fctl.BoolToString(policy.GetProtected())},
		{"Created At", time.Time{Time: policy.GetCreatedAt()}.String()},
		{"Updated At", time.Time{Time: policy.GetUpdatedAt()}.String()},
	}

	if scopes := policy.GetScopes(); len(scopes) > 0 {
		data = append(data, []string{"Scopes", ""})
		for _, scope := range scopes {
			data = append(data, []string{
				"",
				fmt.Sprintf("  - %s (ID: %d)", scope.GetLabel(), scope.GetID()),
			})
			if app := scope.GetApplicationID(); app != nil {
				data = append(data, []string{"", fmt.Sprintf("    Application: %s", *app)})
			}
			if desc := scope.GetDescription(); desc != nil && *desc != "" {
				data = append(data, []string{"", fmt.Sprintf("    Description: %s", *desc)})
			}
		}
	}

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(data).
		Render()
}
