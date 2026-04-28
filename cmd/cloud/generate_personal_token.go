package cloud

import (
	"fmt"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type GeneratePersonalTokenStore struct {
	Token string `json:"token"`
}
type GeneratePersonalTokenController struct {
	store *GeneratePersonalTokenStore
}

var _ fctl.Controller[*GeneratePersonalTokenStore] = (*GeneratePersonalTokenController)(nil)

func NewDefaultGeneratePersonalTokenStore() *GeneratePersonalTokenStore {
	return &GeneratePersonalTokenStore{}
}

func NewGeneratePersonalTokenController() *GeneratePersonalTokenController {
	return &GeneratePersonalTokenController{
		store: NewDefaultGeneratePersonalTokenStore(),
	}
}

func NewGeneratePersonalTokenCommand() *cobra.Command {
	return fctl.NewStackCommand("generate-personal-token",
		fctl.WithAliases("gpt"),
		fctl.WithShortDescription("Generate a personal bearer token"),
		fctl.WithDescription("Generate a personal bearer token"),
		fctl.WithController[*GeneratePersonalTokenStore](NewGeneratePersonalTokenController()),
	)
}

func (c *GeneratePersonalTokenController) GetStore() *GeneratePersonalTokenStore {
	return c.store
}

func (c *GeneratePersonalTokenController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	access, _, err := fctl.EnsureStackAccess(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
		organizationID,
		stackID,
	)
	if err != nil {
		return nil, err
	}
	stackAccess := profile.RootTokens.ID.Claims.
		GetOrganizationAccess(organizationID).
		GetStackAccess(stackID)

	token, err := fctl.FetchStackToken(cmd.Context(), relyingParty.HttpClient(), stackAccess.URI, access.Token)
	if err != nil {
		return nil, err
	}

	c.store.Token = token.AccessToken

	return c, nil
}

func (c *GeneratePersonalTokenController) Render(cmd *cobra.Command, _ []string) error {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), c.store.Token)
	return nil
}
