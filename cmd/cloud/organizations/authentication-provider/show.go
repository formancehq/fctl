package authentication_provider

import (
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type Show struct {
	provider *membershipclient.AuthenticationProviderResponseData
}
type ShowController struct {
	store *Show
}

var _ fctl.Controller[*Show] = (*ShowController)(nil)

func NewDefaultShow() *Show {
	return &Show{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShow(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand(`show`,
		fctl.WithShortDescription("Show authorization provider of organization"),
		fctl.WithController(NewShowController()),
	)
}

func (c *ShowController) GetStore() *Show {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	store := fctl.GetMembershipStore(cmd.Context())
	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	res, _, err := store.Client().
		ReadAuthenticationProvider(cmd.Context(), organizationID).
		Execute()
	if err != nil {
		return nil, err
	}
	c.store.provider = res.Data

	return c, nil
}

func (c *ShowController) Render(_ *cobra.Command, _ []string) error {
	data := [][]string{
		{"Provider", c.store.provider.Type},
		{"Name", c.store.provider.Name},
		{"Client ID", c.store.provider.ClientID},
		{"Client secret", c.store.provider.ClientSecret},
		{"Created at", c.store.provider.CreatedAt.String()},
		{"Updated at", c.store.provider.UpdatedAt.String()},
		{"Redirect URI", c.store.provider.RedirectURI},
	}
	return pterm.DefaultTable.WithHasHeader().WithData(data).Render()
}
