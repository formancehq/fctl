package users

import (
	"fmt"
	"github.com/formancehq/go-libs/pointer"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type LinkStore struct {
	Id      string `json:"id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}
type LinkController struct {
	store *LinkStore
}

var _ fctl.Controller[*LinkStore] = (*LinkController)(nil)

func NewDefaultLinkStore() *LinkStore {
	return &LinkStore{}
}

func NewLinkController() *LinkController {
	return &LinkController{
		store: NewDefaultLinkStore(),
	}
}

func NewLinkCommand() *cobra.Command {
	return fctl.NewCommand("link <user-id>",
		fctl.WithIntFlag("policy-id", 0, "Policy ID"),
		fctl.WithShortDescription("Link user to an organization with properties"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewLinkController()),
	)
}

func (c *LinkController) GetStore() *LinkStore {
	return c.store
}

func (c *LinkController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	policyID := fctl.GetInt(cmd, "policy-id")
	if policyID == 0 {
		return nil, fmt.Errorf("policy id is required")
	}

	req := membershipclient.UpdateOrganizationUserRequest{
		PolicyID: pointer.For(int32(policyID)),
	}
	response, err := store.DefaultAPI.UpsertOrganizationUser(
		cmd.Context(),
		organizationID,
		args[0]).
		UpdateOrganizationUserRequest(req).Execute()
	if err != nil {
		return nil, err
	}

	if response.StatusCode > 300 {
		return nil, fmt.Errorf("error updating user: %s", response.Status)
	}

	return c, nil
}

func (c *LinkController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("User Addd.")
	return nil
}
