package balances

import (
	"github.com/formancehq/fctl/cmd/wallets/internal"
	"github.com/formancehq/fctl/cmd/wallets/internal/views"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ShowStore struct {
	Balance shared.BalanceWithAssets `json:"balance"`
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
	return fctl.NewCommand("show <balance-name>",
		fctl.WithShortDescription("Show a balance"),
		fctl.WithAliases("sh"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		internal.WithTargetingWalletByID(),
		internal.WithTargetingWalletByName(),
		fctl.WithController[*ShowStore](NewShowController()),
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

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	walletID, err := internal.RequireWalletID(cmd, stackClient)
	if err != nil {
		return nil, err
	}

	request := operations.GetBalanceRequest{
		ID:          walletID,
		BalanceName: args[0],
	}
	response, err := stackClient.Wallets.V1.GetBalance(cmd.Context(), request)
	if err != nil {
		return nil, errors.Wrap(err, "getting balance")
	}

	c.store.Balance = response.GetBalanceResponse.Data

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	return views.PrintBalance(cmd.OutOrStdout(), c.store.Balance)
}
