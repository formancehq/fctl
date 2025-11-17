package users

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
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
		fctl.WithStringFlag("role", "", "Roles: (ADMIN, GUEST)"),
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
	store := fctl.GetMembershipStackStore(cmd.Context())

	role := membershipclient.Role(fctl.GetString(cmd, "role"))
	req := membershipclient.UpdateStackUserRequest{}
	if role != "" {
		req.Role = role
	} else {
		return nil, fmt.Errorf("role is required")
	}

	_, err := store.Client().
		UpsertStackUserAccess(cmd.Context(), store.OrganizationId(), store.StackId(), args[0]).
		UpdateStackUserRequest(req).Execute()
	if err != nil {
		return nil, err
	}

	c.store.OrganizationID = store.OrganizationId()
	c.store.StackID = args[0]
	c.store.UserID = args[1]

	return c, nil
}

func (c *LinkController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Organization '%s': stack %s, access roles updated for user %s", c.store.OrganizationID, c.store.StackID, c.store.UserID)

	return nil

}
