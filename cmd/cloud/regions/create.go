package regions

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/membershipclient/v3/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateStore struct {
	RegionId string `json:"regionId"`
	Secret   string `json:"secret"`
}
type CreateController struct {
	store *CreateStore
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand("create [name]",
		fctl.WithAliases("sh", "s"),
		fctl.WithShortDescription("Show region details"),
		fctl.WithArgs(cobra.RangeArgs(0, 1)),
		fctl.WithController[*CreateStore](NewCreateController()),
	)
}

func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	} else {
		name, err = pterm.DefaultInteractiveTextInput.WithMultiLine(false).Show("Enter a name")
		if err != nil {
			return nil, err
		}
	}

	reqBody := components.CreatePrivateRegionRequest{
		Name: name,
	}

	request := operations.CreatePrivateRegionRequest{
		OrganizationID: organizationID,
		Body:           &reqBody,
	}

	response, err := apiClient.CreatePrivateRegion(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.CreatedPrivateRegionResponse == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	region := response.CreatedPrivateRegionResponse.GetData()
	c.store.RegionId = region.GetID()

	secret := region.GetSecret()
	if secret != nil && secret.GetClear() != nil {
		c.store.Secret = *secret.GetClear()
	}

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln(
		"Region created successfully with ID: %s", c.store.RegionId)
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln(
		"Your secret is (keep it safe, we will not be able to give it to you again): %s", c.store.Secret)

	return nil
}
