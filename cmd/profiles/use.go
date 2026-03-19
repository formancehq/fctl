package profiles

import (
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UseStore struct {
}
type UseController struct {
	store *UseStore
}

var _ fctl.Controller[*UseStore] = (*UseController)(nil)

func NewDefaultProfilesUseStore() *UseStore {
	return &UseStore{}
}

func NewProfilesUseController() *UseController {
	return &UseController{
		store: NewDefaultProfilesUseStore(),
	}
}

func ProfileNamesAutoCompletion(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ret, err := fctl.ListProfiles(cmd, func(s string) bool {
		return strings.HasPrefix(s, toComplete)
	})
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	return ret, cobra.ShellCompDirectiveNoFileComp
}

func (c *UseController) GetStore() *UseStore {
	return c.store
}

func (c *UseController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	config, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	_, err = fctl.LoadProfile(cmd, args[0])
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("profile %s not found", args[0])
	}
	if err != nil {
		return nil, err
	}

	config.CurrentProfile = args[0]
	if err := fctl.WriteConfig(cmd, *config); err != nil {
		return nil, fmt.Errorf("Updating config: %w", err)
	}

	return c, nil
}

func (c *UseController) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Selected profile updated!")
	return nil
}

func NewUseCommand() *cobra.Command {
	return fctl.NewCommand("use <name>",
		fctl.WithAliases("u"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithShortDescription("Use profile"),
		fctl.WithValidArgsFunction(ProfileNamesAutoCompletion),
		fctl.WithController(NewProfilesUseController()),
	)
}
