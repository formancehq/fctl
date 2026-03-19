package pools

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type BalancesStore struct {
	Balances *shared.PoolBalances `json:"balances"`
}

type BalancesController struct {
	store *BalancesStore
}

var _ fctl.Controller[*BalancesStore] = (*BalancesController)(nil)

func NewBalancesStore() *BalancesStore {
	return &BalancesStore{
		Balances: &shared.PoolBalances{},
	}
}

func NewBalancesController() *BalancesController {
	return &BalancesController{
		store: NewBalancesStore(),
	}
}

func (c *BalancesController) GetStore() *BalancesStore {
	return c.store
}

func (c *BalancesController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	at, err := time.Parse(time.RFC3339, args[1])
	if err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V1.GetPoolBalances(
		cmd.Context(),
		operations.GetPoolBalancesRequest{
			At:     at,
			PoolID: args[0],
		},
	)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Balances = &response.PoolBalancesResponse.Data

	return c, nil
}

func (c *BalancesController) Render(cmd *cobra.Command, args []string) error {
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

func NewBalancesCommand() *cobra.Command {
	c := NewBalancesController()
	return fctl.NewCommand("balances <poolID> <at>",
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("List pool balances"),
		fctl.WithController[*BalancesStore](c),
	)
}
