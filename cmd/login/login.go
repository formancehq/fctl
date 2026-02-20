package login

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/v2/pkg/oidc"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Store struct {
	profile *fctl.Profile
}
type Controller struct {
	store *Store
}

func NewDefaultStore() *Store {
	return &Store{
		profile: nil,
	}
}
func (c *Controller) GetStore() *Store {
	return c.store
}

func NewLoginController() *Controller {
	return &Controller{
		store: NewDefaultStore(),
	}
}

func (c *Controller) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	membershipUri, err := cmd.Flags().GetString(fctl.MembershipURIFlag)
	if err != nil {
		return nil, err
	}
	if membershipUri == "" {
		membershipUri = profile.GetMembershipURI()
	} else {
		profile.MembershipURI = membershipUri
	}

	relyingParty, err := fctl.GetAuthRelyingParty(cmd.Context(), fctl.GetHttpClient(cmd), membershipUri)
	if err != nil {
		return nil, err
	}

	c.store.profile = profile

	ret, err := fctl.Authenticate(
		cmd.Context(),
		relyingParty,
		fctl.NewPTermDialog(),
		[]fctl.AuthenticationOption{
			fctl.AuthenticateWithScopes(
				oidc.ScopeOpenID,
				oidc.ScopeOfflineAccess,
				"accesses",
				"on_behalf",
			),
			fctl.AuthenticateWithPrompt("no-org"),
		},
		[]fctl.TokenOption{},
	)
	if err != nil {
		return nil, err
	}
	profile.UpdateRootToken(ret)

	currentProfileName := profileName

	cfg.CurrentProfile = currentProfileName
	if err := fctl.WriteConfig(cmd, *cfg); err != nil {
		return nil, err
	}

	if err := fctl.WriteProfile(cmd, currentProfileName, *profile); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Controller) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Logged!")
	return nil
}

func NewCommand() *cobra.Command {
	return fctl.NewCommand("login",
		fctl.WithStringFlag(fctl.MembershipURIFlag, "", "service url"),
		fctl.WithShortDescription("Login"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*Store](NewLoginController()),
	)
}
