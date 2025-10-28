package profiles

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ProfilesRenameStore struct {
	Success bool `json:"success"`
}
type ProfilesRenameController struct {
	store *ProfilesRenameStore
}

var _ fctl.Controller[*ProfilesRenameStore] = (*ProfilesRenameController)(nil)

func NewDefaultProfilesRenameStore() *ProfilesRenameStore {
	return &ProfilesRenameStore{
		Success: false,
	}
}

func NewProfilesRenameController() *ProfilesRenameController {
	return &ProfilesRenameController{
		store: NewDefaultProfilesRenameStore(),
	}
}

func (c *ProfilesRenameController) GetStore() *ProfilesRenameStore {
	return c.store
}

func (c *ProfilesRenameController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	oldName := args[0]
	newName := args[1]

	config, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, err := fctl.LoadProfile(cmd, oldName)
	if err != nil {
		return nil, err
	}

	if err := fctl.DeleteProfile(cmd, oldName); err != nil {
		return nil, err
	}

	if err := fctl.WriteProfile(cmd, newName, *profile); err != nil {
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

func (c *ProfilesRenameController) Render(cmd *cobra.Command, args []string) error {
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
