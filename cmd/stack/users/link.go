package users

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type LinkStore struct {
	OrganizationID string `json:"organizationId"`
	UserID         string `json:"userId"`
	StackID        string `json:"stackId"`
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
		fctl.WithIntFlag("policy-id", 0, "Policy id"),
		fctl.WithShortDescription("Link stack user with properties"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*LinkStore](NewLinkController()),
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
	req := membershipclient.UpdateStackUserRequest{}
	if policyID != 0 {
		req.PolicyID = int32(policyID)
	} else {
		return nil, fmt.Errorf("policy id is required")
	}

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	_, err = store.DefaultAPI.
		UpsertStackUserAccess(cmd.Context(), organizationID, stackID, args[0]).
		UpdateStackUserRequest(req).Execute()
	if err != nil {
		return nil, err
	}

	c.store.OrganizationID = organizationID
	c.store.StackID = args[0]
	c.store.UserID = args[1]

	return c, nil
}

func (c *LinkController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Organization '%s': stack %s, access roles updated for user %s", c.store.OrganizationID, c.store.StackID, c.store.UserID)

	return nil

}
