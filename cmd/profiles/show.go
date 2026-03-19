package profiles

import (
	"errors"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ProfilesShowStore struct {
	MembershipURI       string `json:"membershipUri"`
	DefaultOrganization string `json:"defaultOrganization"`
	DefaultStack        string `json:"defaultStack"`
}
type ProfilesShowController struct {
	store *ProfilesShowStore
}

var _ fctl.Controller[*ProfilesShowStore] = (*ProfilesShowController)(nil)

func NewDefaultProfilesShowStore() *ProfilesShowStore {
	return &ProfilesShowStore{
		MembershipURI:       "",
		DefaultOrganization: "",
		DefaultStack:        "",
	}
}

func NewProfilesShowController() *ProfilesShowController {
	return &ProfilesShowController{
		store: NewDefaultProfilesShowStore(),
	}
}

func (c *ProfilesShowController) GetStore() *ProfilesShowStore {
	return c.store
}

func (c *ProfilesShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	p, err := fctl.LoadProfile(cmd, args[0])
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("not found")
	}

	c.store.DefaultOrganization = p.GetDefaultOrganization()
	c.store.MembershipURI = p.GetMembershipURI()
	c.store.DefaultStack = p.GetDefaultStack()

	return c, nil
}

func (c *ProfilesShowController) Render(cmd *cobra.Command, args []string) error {

	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("Membership URI"), c.store.MembershipURI})
	tableData = append(tableData, []string{pterm.LightCyan("Default organization"), c.store.DefaultOrganization})
	tableData = append(tableData, []string{pterm.LightCyan("Default stack"), c.store.DefaultStack})
	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("show <name>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithAliases("s"),
		fctl.WithShortDescription("Show profile"),
		fctl.WithValidArgsFunction(ProfileNamesAutoCompletion),
		fctl.WithController(NewProfilesShowController()),
	)
}
