package profiles

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ProfilesDeleteStore struct {
	Success bool `json:"success"`
}
type ProfileDeleteController struct {
	store *ProfilesDeleteStore
}

var _ fctl.Controller[*ProfilesDeleteStore] = (*ProfileDeleteController)(nil)

func NewDefaultDeleteProfileStore() *ProfilesDeleteStore {
	return &ProfilesDeleteStore{
		Success: false,
	}
}

func NewDeleteProfileController() *ProfileDeleteController {
	return &ProfileDeleteController{
		store: NewDefaultDeleteProfileStore(),
	}
}

func (c *ProfileDeleteController) GetStore() *ProfilesDeleteStore {
	return c.store
}

func (c *ProfileDeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if err := fctl.DeleteProfile(cmd, args[0]); err != nil {
		return nil, err
	}

	c.store.Success = true

	return c, nil
}

func (c *ProfileDeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Profile deleted!")
	return nil
}

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete <name>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithShortDescription("Delete a profile"),
		fctl.WithValidArgsFunction(ProfileNamesAutoCompletion),
		fctl.WithController(NewDeleteProfileController()),
	)
}
