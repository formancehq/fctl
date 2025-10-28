package users

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ShowStore struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	PolicyID int32  `json:"policyID"`
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
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController(NewShowController()),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	userResponse, _, err := store.DefaultAPI.ReadUserOfOrganization(cmd.Context(), organizationID, args[0]).Execute()
	if err != nil {
		return nil, err
	}

	c.store.Id = userResponse.Data.Id
	c.store.Email = userResponse.Data.Email
	c.store.PolicyID = userResponse.Data.PolicyID

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.Id})
	tableData = append(tableData, []string{pterm.LightCyan("Email"), c.store.Email})
	tableData = append(tableData, []string{
		pterm.LightCyan("Role"),
		pterm.LightCyan(c.store.PolicyID),
	})

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()

}
