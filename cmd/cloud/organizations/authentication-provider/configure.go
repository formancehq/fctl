package authentication_provider

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
	fctl "github.com/formancehq/fctl/pkg"
)

type Configure struct {
	RedirectURI string `json:"redirectURI"`
}
type ConfigureController struct {
	store *Configure
}

var _ fctl.Controller[*Configure] = (*ConfigureController)(nil)

func NewDefaultConfigure() *Configure {
	return &Configure{}
}

func NewConfigureController() *ConfigureController {
	return &ConfigureController{
		store: NewDefaultConfigure(),
	}
}

func NewConfigureCommand() *cobra.Command {
	return fctl.NewCommand(`configure <type> <name> <client-id> <client-secret>`,
		fctl.WithValidArgs("github", "google", "microsoft", "oidc"),
		fctl.WithArgs(cobra.ExactArgs(4)),
		fctl.WithShortDescription("Configure authorization provider for organization"),
		fctl.WithController(NewConfigureController()),
		fctl.WithStringFlag("oidc-issuer", "", "Used when type = oidc"),
		fctl.WithStringFlag("microsoft-tenant", "tenant", "Used when type = microsoft"),
	)
}

func (c *ConfigureController) GetStore() *Configure {
	return c.store
}

func (c *ConfigureController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	var requestData components.AuthenticationProviderData

	switch args[0] {
	case "github":
		config := components.AuthenticationProviderDataGithubIDPConfig{
			Type:         components.AuthenticationProviderDataGithubIDPConfigTypeGithub,
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
			Config:       components.AuthenticationProviderDataGithubIDPConfigConfig{},
		}
		requestData = components.CreateAuthenticationProviderDataAuthenticationProviderDataGithubIDPConfig(config)
	case "google":
		config := components.AuthenticationProviderDataGoogleIDPConfig{
			Type:         components.AuthenticationProviderDataGoogleIDPConfigTypeGoogle,
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
			Config:       components.AuthenticationProviderDataGoogleIDPConfigConfig{},
		}
		requestData = components.CreateAuthenticationProviderDataAuthenticationProviderDataGoogleIDPConfig(config)
	case "microsoft":
		config := components.AuthenticationProviderDataMicrosoftIDPConfig{
			Type:         components.AuthenticationProviderDataMicrosoftIDPConfigTypeMicrosoft,
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
			Config: components.AuthenticationProviderDataMicrosoftIDPConfigConfig{
				Tenant: pointer.For(fctl.GetString(cmd, "microsoft-tenant")),
			},
		}
		requestData = components.CreateAuthenticationProviderDataAuthenticationProviderDataMicrosoftIDPConfig(config)
	case "oidc":
		config := components.AuthenticationProviderDataOIDCConfig{
			Type:         components.AuthenticationProviderDataOIDCConfigTypeOidc,
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
			Config: components.AuthenticationProviderDataOIDCConfigConfig{
				Issuer: fctl.GetString(cmd, "oidc-issuer"),
			},
		}
		requestData = components.CreateAuthenticationProviderDataAuthenticationProviderDataOIDCConfig(config)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", args[0])
	}

	request := operations.UpsertAuthenticationProviderRequest{
		OrganizationID: organizationID,
		Body:           &requestData,
	}

	response, err := apiClient.UpsertAuthenticationProvider(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.AuthenticationProviderResponse == nil || response.AuthenticationProviderResponse.GetData() == nil {
		return nil, fmt.Errorf("unexpected response: no data")
	}

	data := response.AuthenticationProviderResponse.GetData()
	switch data.Type {
	case components.DataTypeAuthenticationProviderResponseGithubIDPConfig:
		if p := data.AuthenticationProviderResponseGithubIDPConfig; p != nil {
			c.store.RedirectURI = p.GetRedirectURI()
		}
	case components.DataTypeAuthenticationProviderResponseGoogleIDPConfig:
		if p := data.AuthenticationProviderResponseGoogleIDPConfig; p != nil {
			c.store.RedirectURI = p.GetRedirectURI()
		}
	case components.DataTypeAuthenticationProviderResponseMicrosoftIDPConfig:
		if p := data.AuthenticationProviderResponseMicrosoftIDPConfig; p != nil {
			c.store.RedirectURI = p.GetRedirectURI()
		}
	case components.DataTypeAuthenticationProviderResponseOIDCConfig:
		if p := data.AuthenticationProviderResponseOIDCConfig; p != nil {
			c.store.RedirectURI = p.GetRedirectURI()
		}
	}

	return c, nil
}

func (c *ConfigureController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Authorization provider configured successfully")
	pterm.Info.Println(fmt.Sprintf("Redirect URI: %s", c.store.RedirectURI))

	return nil
}
