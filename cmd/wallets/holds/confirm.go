package holds

import (
	"fmt"
	"math/big"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/pkg"
)

type ConfirmStore struct {
	Success bool   `json:"success"`
	HoldId  string `json:"holdId"`
}
type ConfirmController struct {
	store      *ConfirmStore
	finalFlag  string
	amountFlag string
	ikFlag     string
}

var _ fctl.Controller[*ConfirmStore] = (*ConfirmController)(nil)

func NewDefaultConfirmStore() *ConfirmStore {
	return &ConfirmStore{}
}

func NewConfirmController() *ConfirmController {
	return &ConfirmController{
		store:      NewDefaultConfirmStore(),
		finalFlag:  "final",
		amountFlag: "amount",
		ikFlag:     "ik",
	}
}

func NewConfirmCommand() *cobra.Command {
	c := NewConfirmController()
	return fctl.NewCommand("confirm <hold-id>",
		fctl.WithShortDescription("Confirm a hold"),
		fctl.WithAliases("c", "conf"),
		fctl.WithArgs(cobra.RangeArgs(1, 2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithBoolFlag(c.finalFlag, false, "Is final debit (close hold)"),
		fctl.WithStringFlag(c.ikFlag, "", "Idempotency Key"),
		fctl.WithIntFlag(c.amountFlag, 0, "Amount to confirm"),
		fctl.WithController[*ConfirmStore](c),
	)
}

func (c *ConfirmController) GetStore() *ConfirmStore {
	return c.store
}

func (c *ConfirmController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	final := fctl.GetBool(cmd, c.finalFlag)
	amount := int64(fctl.GetInt(cmd, c.amountFlag))

	request := operations.ConfirmHoldRequest{
		HoldID: args[0],
		ConfirmHoldRequest: &shared.ConfirmHoldRequest{
			Amount: big.NewInt(amount),
			Final:  &final,
		},
	}
	_, err = stackClient.Wallets.V1.ConfirmHold(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("confirming hold: %w", err)
	}

	c.store.Success = true //Todo: check status code
	c.store.HoldId = args[0]

	return c, nil
}

func (c *ConfirmController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Hold '%s' confirmed!", args[0])

	return nil

}
