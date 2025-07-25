package users

import (
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ShowStore struct {
	Id    string                `json:"id"`
	Email string                `json:"email"`
	Role  membershipclient.Role `json:"role"`
}
type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewDefaultShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("show <user-id>",
		fctl.WithAliases("s"),
		fctl.WithShortDescription("Show user by id"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController(NewShowController()),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetMembershipStore(cmd.Context())
	organizationID, err := fctl.ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return nil, err
	}

	userResponse, _, err := store.Client().ReadUserOfOrganization(cmd.Context(), organizationID, args[0]).Execute()
	if err != nil {
		return nil, err
	}

	c.store.Id = userResponse.Data.Id
	c.store.Email = userResponse.Data.Email
	c.store.Role = userResponse.Data.Role

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.Id})
	tableData = append(tableData, []string{pterm.LightCyan("Email"), c.store.Email})
	tableData = append(tableData, []string{
		pterm.LightCyan("Role"),
		pterm.LightCyan(c.store.Role),
	})

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
