package pools

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
)

type LatestBalancesStore struct {
	Balances *shared.PoolBalances `json:"balances"`
}

type LatestBalancesController struct {
	PaymentsVersion versions.Version

	store *LatestBalancesStore
}

func (c *LatestBalancesController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*LatestBalancesStore] = (*LatestBalancesController)(nil)

func NewLatestBalancesStore() *LatestBalancesStore {
	return &LatestBalancesStore{
		Balances: &shared.PoolBalances{},
	}
}

func NewLatestBalancesController() *LatestBalancesController {
	return &LatestBalancesController{
		store: NewLatestBalancesStore(),
	}
}

func (c *LatestBalancesController) GetStore() *LatestBalancesStore {
	return c.store
}

func (c *LatestBalancesController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	poolID := args[0]

	switch c.PaymentsVersion {
	case versions.V1, versions.V2:
		return c.CallV1(cmd.Context(), store, poolID)
	case versions.V3:
		return c.CallV3(cmd.Context(), store, poolID)
	}

	return nil, fmt.Errorf("unsupported payments version: %d", c.PaymentsVersion)
}

func (c *LatestBalancesController) CallV1(context context.Context, store *fctl.StackStore, poolID string) (fctl.Renderable, error) {
	response, err := store.Client().Payments.V1.GetPoolBalancesLatest(
		context,
		operations.GetPoolBalancesLatestRequest{
			PoolID: poolID,
		})
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	c.store.Balances = &response.PoolBalancesResponse.Data
	return c, nil
}

func (c *LatestBalancesController) CallV3(context context.Context, store *fctl.StackStore, poolID string) (fctl.Renderable, error) {
	response, err := store.Client().Payments.V3.GetPoolBalancesLatest(
		context,
		operations.V3GetPoolBalancesLatestRequest{
			PoolID: poolID,
		})
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	v3Balances := &response.V3PoolBalancesResponse.Data

	poolBalances := make([]shared.PoolBalance, len(*v3Balances))
	for _, v3Balance := range *v3Balances {
		poolBalance := shared.PoolBalance{
			Asset:  v3Balance.Asset,
			Amount: v3Balance.Amount,
		}
		poolBalances = append(poolBalances, poolBalance)
	}
	c.store.Balances = &shared.PoolBalances{Balances: poolBalances}
	return c, nil
}

func (c *LatestBalancesController) Render(cmd *cobra.Command, args []string) error {
	tableData := fctl.Map(c.store.Balances.Balances, func(balance shared.PoolBalance) []string {
		return []string{
			balance.Asset,
			balance.Amount.String(),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"Asset", "Amount"})
	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}

func NewLatestBalancesCommand() *cobra.Command {
	c := NewLatestBalancesController()
	return fctl.NewCommand("balances <poolID>",
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("List pool latest balances"),
		fctl.WithController[*LatestBalancesStore](c),
	)
}
