package users

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type ShowStore struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	PolicyID int64  `json:"policyID"`
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
	return fctl.NewCommand("show <user-id>",
		fctl.WithAliases("s"),
		fctl.WithShortDescription("Show user by id"),
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

	request := operations.ReadUserOfOrganizationRequest{
		OrganizationID: organizationID,
		UserID:         args[0],
	}

	response, err := apiClient.ReadUserOfOrganization(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.ReadOrganizationUserResponse == nil || response.ReadOrganizationUserResponse.GetData() == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	data := response.ReadOrganizationUserResponse.GetData()
	c.store.Id = data.GetID()
	c.store.Email = data.GetEmail()
	c.store.PolicyID = data.GetPolicyID()

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.Id})
	tableData = append(tableData, []string{pterm.LightCyan("Email"), c.store.Email})
	tableData = append(tableData, []string{
		pterm.LightCyan("Role"),
		pterm.LightCyan(c.store.PolicyID),
	})

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
