package profiles

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/go-libs/collectionutils"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
	cfg, err := fctl.GetConfig(cmd)
	if err != nil {
		return nil, err
	}

	fctl.GetCurrentProfile(cmd, cfg).SetDefaultOrganization(args[0])
	fctl.GetCurrentProfile(cmd, cfg).SetDefaultStack("")
	if err := cfg.Persist(); err != nil {
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

func organizationCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := fctl.NewMembershipStore(cmd); err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	mbStore := fctl.GetMembershipStore(cmd.Context())

	ret, res, err := mbStore.Client().ListOrganizations(cmd.Context()).Execute()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	if res.StatusCode > 300 {
		return []string{}, cobra.ShellCompDirectiveError
	}

	opts := collectionutils.Reduce(ret.Data, func(acc []string, o membershipclient.OrganizationExpanded) []string {
		return append(acc, fmt.Sprintf("%s\t%s", o.Id, o.Name))
	}, []string{})

	return opts, cobra.ShellCompDirectiveDefault
}
