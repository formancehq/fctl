package authentication_provider

import (
	"fmt"

	"github.com/formancehq/go-libs/pointer"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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

	requestData := membershipclient.AuthenticationProviderData{}

	switch args[0] {
	case "github":
		requestData.GithubIDPConfig = &membershipclient.GithubIDPConfig{
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
		}
	case "google":
		requestData.GoogleIDPConfig = &membershipclient.GoogleIDPConfig{
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
		}
	case "microsoft":
		requestData.MicrosoftIDPConfig = &membershipclient.MicrosoftIDPConfig{
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
			Config: membershipclient.MicrosoftIDPConfigAllOfConfig{
				Tenant: pointer.For(fctl.GetString(cmd, "microsoft-tenant")),
			},
		}
	case "oidc":
		requestData.OIDCConfig = &membershipclient.OIDCConfig{
			Name:         args[1],
			ClientID:     args[2],
			ClientSecret: args[3],
			Config: membershipclient.OIDCConfigAllOfConfig{
				Issuer: fctl.GetString(cmd, "oidc-issuer"),
			},
		}
	}

	res, _, err := store.DefaultAPI.
		UpsertAuthenticationProvider(cmd.Context(), organizationID).
		Body(requestData).
		Execute()
	if err != nil {
		return nil, err
	}
	c.store.RedirectURI = res.Data.RedirectURI

	return c, nil
}

func (c *ConfigureController) Render(_ *cobra.Command, _ []string) error {
	pterm.Success.Println("Authorization provider configured successfully")
	pterm.Info.Println(fmt.Sprintf("Redirect URI: %s", c.store.RedirectURI))

	return nil
}
