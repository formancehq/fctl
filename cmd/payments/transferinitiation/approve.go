package transferinitiation

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
)

type ApproveStore struct {
	TransferID string `json:"transferId"`
	TaskID     string `json:"taskId"`
}
type ApproveController struct {
	PaymentsVersion versions.Version

	store *ApproveStore
}

func (c *ApproveController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ApproveStore] = (*ApproveController)(nil)

func NewApproveStore() *ApproveStore {
	return &ApproveStore{}
}

func NewApproveController() *ApproveController {
	return &ApproveController{
		store: NewApproveStore(),
	}
}

func NewApproveCommand() *cobra.Command {
	c := NewApproveController()
	return fctl.NewCommand("approve <transferInitiationID>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Approve a transfer initiation"),
		fctl.WithAliases("a"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*ApproveStore](c),
	)
}

func (c *ApproveController) GetStore() *ApproveStore {
	return c.store
}

func (c *ApproveController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V3 {
		return nil, fmt.Errorf("transfer initiation approval is only supported in >= v3.0.0")
	}

	if !fctl.CheckStackApprobation(cmd, store.Stack(), "You are about to approve the transfer initiation %q", args[0]) {
		return nil, fctl.ErrMissingApproval
	}

	response, err := store.Client().Payments.V3.ApprovePaymentInitiation(cmd.Context(), operations.V3ApprovePaymentInitiationRequest{
		PaymentInitiationID: args[0],
	})
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	c.store.TransferID = args[0]
	c.store.TaskID = response.V3ApprovePaymentInitiationResponse.Data.TaskID

	return c, nil
}

func (c *ApproveController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Transfer Initiation scheduled with TaskID %q", c.store.TaskID)

	return nil
}
