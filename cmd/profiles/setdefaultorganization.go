package profiles

import (
	"fmt"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/collectionutils"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"strings"
)

type ProfilesSetDefaultOrganizationStore struct {
	Success bool `json:"success"`
}
type ProfilesSetDefaultOrganizationController struct {
	store *ProfilesSetDefaultOrganizationStore
}

var _ fctl.Controller[*ProfilesSetDefaultOrganizationStore] = (*ProfilesSetDefaultOrganizationController)(nil)

func NewDefaultProfilesSetDefaultOrganizationStore() *ProfilesSetDefaultOrganizationStore {
	return &ProfilesSetDefaultOrganizationStore{
		Success: false,
	}
}

func NewProfilesSetDefaultOrganizationController() *ProfilesSetDefaultOrganizationController {
	return &ProfilesSetDefaultOrganizationController{
		store: NewDefaultProfilesSetDefaultOrganizationStore(),
	}
}

func (c *ProfilesSetDefaultOrganizationController) GetStore() *ProfilesSetDefaultOrganizationStore {
	return c.store
}

func (c *ProfilesSetDefaultOrganizationController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	currentProfile, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	currentProfile.DefaultOrganization = args[0]
	currentProfile.DefaultStack = ""

	if err := fctl.WriteProfile(cmd, cfg.CurrentProfile, *currentProfile); err != nil {
		return nil, errors.Wrap(err, "Updating config")
	}

	c.store.Success = true
	return c, nil
}

func (c *ProfilesSetDefaultOrganizationController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Default organization updated!")
	return nil
}

func NewSetDefaultOrganizationCommand() *cobra.Command {
	return fctl.NewCommand("set-default-organization <organization-id>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithAliases("sdo"),
		fctl.WithShortDescription("Set default organization"),
		fctl.WithValidArgsFunction(organizationCompletion),
		fctl.WithController(NewProfilesSetDefaultOrganizationController()),
	)
}

func organizationCompletion(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	profile, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	list := collectionutils.Map(profile.RootTokens.ID.Claims.Organizations, func(from fctl.OrganizationAccess) string {
		return fmt.Sprintf("%s\t%s", from.ID, from.DisplayName)
	})
	list = collectionutils.Filter(list, func(s string) bool {
		return toComplete == "" || strings.HasPrefix(s, toComplete)
	})

	return list, cobra.ShellCompDirectiveNoFileComp
}
