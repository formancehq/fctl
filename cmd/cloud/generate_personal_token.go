package cloud

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
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

func (c *GeneratePersonalTokenController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	store := fctl.GetMembershipStore(cmd.Context())
	profile := fctl.GetCurrentProfile(cmd, store.Config)

	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	stack, err := fctl.ResolveStack(cmd, store.Config, organizationID)
	if err != nil {
		return nil, err
	}

	token, err := profile.GetStackToken(cmd.Context(), fctl.GetHttpClient(cmd, map[string][]string{}), stack)
	if err != nil {
		return nil, err
	}

	c.store.Token = token.AccessToken

	return c, nil
}

func (c *GeneratePersonalTokenController) Render(cmd *cobra.Command, args []string) error {

	fmt.Fprintln(cmd.OutOrStdout(), c.store.Token)
	return nil
}
