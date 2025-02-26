package transferinitiation

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type RejectStore struct {
	TransferID string `json:"transferId"`
}

type RejectController struct {
	PaymentsVersion versions.Version

	store *RejectStore
}

func (c *RejectController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*RejectStore] = (*RejectController)(nil)

func NewRejectStore() *RejectStore {
	return &RejectStore{}
}

func NewRejectController() *RejectController {
	return &RejectController{
		store: NewRejectStore(),
	}
}

func NewRejectCommand() *cobra.Command {
	c := NewRejectController()
	return fctl.NewCommand("reject <transferInitiationID>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Reject a transfer initiation"),
		fctl.WithAliases("rj"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*RejectStore](c),
	)
}

func (c *RejectController) GetStore() *RejectStore {
	return c.store
}

func (c *RejectController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V3 {
		return nil, fmt.Errorf("transfer initiation rejection is only supported in >= v3.0.0")
	}

	if !fctl.CheckStackApprobation(cmd, store.Stack(), "You are about to reject the transfer initiation %q", args[0]) {
		return nil, fctl.ErrMissingApproval
	}

	response, err := store.Client().Payments.V3.RejectPaymentInitiation(cmd.Context(), operations.V3RejectPaymentInitiationRequest{
		PaymentInitiationID: args[0],
	})
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	c.store.TransferID = args[0]

	return c, nil
}

func (c *RejectController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Transfer Initiation %q was rejected", c.store.TransferID)
	return nil
}
