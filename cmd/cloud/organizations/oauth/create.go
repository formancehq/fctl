package oauth

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type Create struct {
	Organization *membershipclient.CreateClientResponseResponse `json:"organization"`
}
type CreateController struct {
	store *Create
}

var _ fctl.Controller[*Create] = (*CreateController)(nil)

func NewDefaultCreate() *Create {
	return &Create{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultCreate(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand(`create`,
		fctl.WithShortDescription("Create organization OAuth client"),
		fctl.WithConfirmFlag(),
		fctl.WithDeprecated("Use `fctl cloud organizations clients create` instead"),
		fctl.WithController(NewCreateController()),
	)
}

func (c *CreateController) GetStore() *Create {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	store, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckOrganizationApprobation(cmd, "You are about to create a new organization OAuth client") {
		return nil, fctl.ErrMissingApproval
	}

	response, _, err := store.DefaultAPI.CreateOrganizationClient(cmd.Context(), organizationID).Execute()
	if err != nil {
		return nil, err
	}

	c.store.Organization = response

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	data := [][]string{
		{"Client ID", fmt.Sprintf("organization_%s", c.store.Organization.Data.Id)},
		{"Client Secret", *c.store.Organization.Data.Secret.Clear},
	}
	pterm.DefaultTable.WithHasHeader().WithData(data).Render()

	return nil
}
