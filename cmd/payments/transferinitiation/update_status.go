package transferinitiation

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type UpdateStatusStore struct {
	TransferID string `json:"transferId"`
	Status     string `json:"status"`
	Success    bool   `json:"success"`
}
type UpdateStatusController struct {
	PaymentsVersion versions.Version

	store *UpdateStatusStore
}

func (c *UpdateStatusController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*UpdateStatusStore] = (*UpdateStatusController)(nil)

func NewUpdateStatusStore() *UpdateStatusStore {
	return &UpdateStatusStore{}
}

func NewUpdateStatusController() *UpdateStatusController {
	return &UpdateStatusController{
		store: NewUpdateStatusStore(),
	}
}

func NewUpdateStatusCommand() *cobra.Command {
	c := NewUpdateStatusController()
	return fctl.NewCommand("update_status <transferID> <status>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Update the status of a transfer initiation (deprecated in >= v3.0.0)"),
		fctl.WithAliases("u"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithController[*UpdateStatusStore](c),
	)
}

func (c *UpdateStatusController) GetStore() *UpdateStatusStore {
	return c.store
}

func (c *UpdateStatusController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V1 {
		return nil, fmt.Errorf("transfer initiation updates are only supported in >= v2.0.0")
	}

	if !fctl.CheckStackApprobation(cmd, store.Stack(), "You are about to update the status of the transfer initiation '%s' to '%s'", args[0], args[1]) {
		return nil, fctl.ErrMissingApproval
	}

	//nolint:gosimple
	response, err := store.Client().Payments.V1.UdpateTransferInitiationStatus(cmd.Context(), operations.UdpateTransferInitiationStatusRequest{
		UpdateTransferInitiationStatusRequest: shared.UpdateTransferInitiationStatusRequest{
			Status: shared.Status(args[1]),
		},
		TransferID: args[0],
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.TransferID = args[0]
	c.store.Status = args[1]
	c.store.Success = true

	return c, nil
}

func (c *UpdateStatusController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Update Transfer Initiation status with ID: %s and status %s", c.store.TransferID, c.store.Status)

	return nil
}
