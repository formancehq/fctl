package profiles

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type RenameStore struct {
	Success bool `json:"success"`
}
type RenameController struct {
	store *RenameStore
}

var _ fctl.Controller[*RenameStore] = (*RenameController)(nil)

func NewDefaultProfilesRenameStore() *RenameStore {
	return &RenameStore{
		Success: false,
	}
}

func NewProfilesRenameController() *RenameController {
	return &RenameController{
		store: NewDefaultProfilesRenameStore(),
	}
}

func (c *RenameController) GetStore() *RenameStore {
	return c.store
}

func (c *RenameController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	oldName := args[0]
	newName := args[1]

	config, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	if err := fctl.RenameProfile(cmd, oldName, newName); err != nil {
		return nil, err
	}

	if config.CurrentProfile == oldName {
		config.CurrentProfile = newName
		if err := fctl.WriteConfig(cmd, *config); err != nil {
			return nil, err
		}
	}

	c.store.Success = true
	return c, nil
}

func (c *RenameController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Profile renamed!")
	return nil
}

func NewRenameCommand() *cobra.Command {
	return fctl.NewCommand("rename <old-name> <new-name>",
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithShortDescription("Rename a profile"),
		fctl.WithValidArgsFunction(ProfileNamesAutoCompletion),
		fctl.WithController(NewProfilesRenameController()),
	)
}
