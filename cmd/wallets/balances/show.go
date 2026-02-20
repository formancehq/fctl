package balances

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/wallets/internal"
	"github.com/formancehq/fctl/v3/cmd/wallets/internal/views"
	fctl "github.com/formancehq/fctl/v3/pkg"
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
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
		return nil, fmt.Errorf("getting balance: %w", err)
	}

	c.store.Balance = response.GetBalanceResponse.Data

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	return views.PrintBalance(cmd.OutOrStdout(), c.store.Balance)
}
