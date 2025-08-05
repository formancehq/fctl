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

type SetDefaultStackStore struct {
	Success bool `json:"success"`
}
type SetDefaultStackController struct {
	store *SetDefaultStackStore
}

var _ fctl.Controller[*SetDefaultStackStore] = (*SetDefaultStackController)(nil)

func NewDefaultSetDefaultStackStore() *SetDefaultStackStore {
	return &SetDefaultStackStore{
		Success: false,
	}
}

func NewSetDefaultStackController() *SetDefaultStackController {
	return &SetDefaultStackController{
		store: NewDefaultSetDefaultStackStore(),
	}
}

func (c *SetDefaultStackController) GetStore() *SetDefaultStackStore {
	return c.store
}

func (c *SetDefaultStackController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.GetConfig(cmd)
	if err != nil {
		return nil, err
	}

	if err := fctl.NewMembershipStore(cmd); err != nil {
		return nil, err
	}

	store := fctl.GetMembershipStore(cmd.Context())
	organizationId, err := fctl.ResolveOrganizationID(cmd, cfg, store.Client())
	if err != nil {
		return nil, err
	}
	stackRes, res, err := store.Client().GetStack(cmd.Context(), organizationId, args[0]).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get stack: %w", err)
	}

	if res.StatusCode != 200 {
		return nil, errors.Errorf("Failed to get stack: %s", res.Status)
	}

	cfg.GetCurrentProfile().SetDefaultStack(stackRes.Data.Id)
	if err := cfg.Persist(); err != nil {
		return nil, errors.Wrap(err, "Updating config")
	}

	c.store.Success = true
	return c, nil
}

func (c *SetDefaultStackController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Default stack updated!")
	return nil
}

func NewSetDefaultStackCommand() *cobra.Command {
	return fctl.NewCommand("set-default-stack <stack-id>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithAliases("sds"),
		fctl.WithShortDescription("Set default stack"),
		fctl.WithValidArgsFunction(stackCompletion),
		fctl.WithController(NewSetDefaultStackController()),
	)
}

func stackCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := fctl.NewMembershipOrganizationStore(cmd); err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	orgStore := fctl.GetOrganizationStore(cmd)

	ret, res, err := orgStore.Client().ListStacks(cmd.Context(), orgStore.OrganizationId()).Execute()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	if res.StatusCode > 300 {
		return []string{}, cobra.ShellCompDirectiveError
	}

	opts := collectionutils.Reduce(ret.Data, func(acc []string, s membershipclient.Stack) []string {
		return append(acc, fmt.Sprintf("%s\t%s", s.Id, s.Name))
	}, []string{})

	return opts, cobra.ShellCompDirectiveNoFileComp
}
