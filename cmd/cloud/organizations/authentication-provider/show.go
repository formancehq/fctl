package authentication_provider

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/v3/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Show struct {
	provider *components.Data
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, apiClient, err := fctl.NewMembershipClientForOrganizationFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	request := operations.ReadAuthenticationProviderRequest{
		OrganizationID: organizationID,
	}

	response, err := apiClient.ReadAuthenticationProvider(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.AuthenticationProviderResponse == nil || response.AuthenticationProviderResponse.GetData() == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	c.store.provider = response.AuthenticationProviderResponse.GetData()

	return c, nil
}

func (c *ShowController) Render(_ *cobra.Command, _ []string) error {
	var providerType, name, clientID, clientSecret, redirectURI string
	var createdAt, updatedAt interface{}

	if c.store.provider == nil {
		return fmt.Errorf("no provider data")
	}

	switch c.store.provider.Type {
	case components.DataTypeAuthenticationProviderResponseGithubIDPConfig:
		if p := c.store.provider.AuthenticationProviderResponseGithubIDPConfig; p != nil {
			providerType = string(p.GetType())
			name = p.GetName()
			clientID = p.GetClientID()
			clientSecret = p.GetClientSecret()
			createdAt = p.GetCreatedAt()
			updatedAt = p.GetUpdatedAt()
			redirectURI = p.GetRedirectURI()
		}
	case components.DataTypeAuthenticationProviderResponseGoogleIDPConfig:
		if p := c.store.provider.AuthenticationProviderResponseGoogleIDPConfig; p != nil {
			providerType = string(p.GetType())
			name = p.GetName()
			clientID = p.GetClientID()
			clientSecret = p.GetClientSecret()
			createdAt = p.GetCreatedAt()
			updatedAt = p.GetUpdatedAt()
			redirectURI = p.GetRedirectURI()
		}
	case components.DataTypeAuthenticationProviderResponseMicrosoftIDPConfig:
		if p := c.store.provider.AuthenticationProviderResponseMicrosoftIDPConfig; p != nil {
			providerType = string(p.GetType())
			name = p.GetName()
			clientID = p.GetClientID()
			clientSecret = p.GetClientSecret()
			createdAt = p.GetCreatedAt()
			updatedAt = p.GetUpdatedAt()
			redirectURI = p.GetRedirectURI()
		}
	case components.DataTypeAuthenticationProviderResponseOIDCConfig:
		if p := c.store.provider.AuthenticationProviderResponseOIDCConfig; p != nil {
			providerType = string(p.GetType())
			name = p.GetName()
			clientID = p.GetClientID()
			clientSecret = p.GetClientSecret()
			createdAt = p.GetCreatedAt()
			updatedAt = p.GetUpdatedAt()
			redirectURI = p.GetRedirectURI()
		}
	}

	data := [][]string{
		{"Provider", providerType},
		{"Name", name},
		{"Client ID", clientID},
		{"Client secret", clientSecret},
		{"Created at", fmt.Sprintf("%v", createdAt)},
		{"Updated at", fmt.Sprintf("%v", updatedAt)},
		{"Redirect URI", redirectURI},
	}
	return pterm.DefaultTable.WithHasHeader().WithData(data).Render()
}
