package psu

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type DeleteStore struct {
	TaskID string `json:"taskID"`
}

type DeleteController struct {
	PaymentsVersion versions.Version
	store           *DeleteStore
}

func (c *DeleteController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*DeleteStore] = (*DeleteController)(nil)

func NewDeleteStore() *DeleteStore {
	return &DeleteStore{}
}

func NewDeleteController() *DeleteController {
	return &DeleteController{
		store: NewDeleteStore(),
	}
}

func NewDeleteCommand() *cobra.Command {
	c := NewDeleteController()
	return fctl.NewCommand("delete <paymentServiceUserID>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Delete a payment service user"),
		fctl.WithAliases("del", "rm"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*DeleteStore](c),
	)
}

func (c *DeleteController) GetStore() *DeleteStore {
	return c.store
}

func (c *DeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion.Major < versions.V3 {
		return nil, fmt.Errorf("payment service users require Payments API v3")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to delete payment service user '%s'", args[0]) {
		return nil, fctl.ErrMissingApproval
	}

	response, err := stackClient.Payments.V3.DeletePaymentServiceUser(cmd.Context(), operations.V3DeletePaymentServiceUserRequest{
		PaymentServiceUserID: args[0],
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	if response.V3PaymentServiceUserDeleteResponse == nil {
		return nil, fmt.Errorf("unexpected empty response")
	}

	c.store.TaskID = response.V3PaymentServiceUserDeleteResponse.Data.TaskID

	return c, nil
}

func (c *DeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Payment service user deletion scheduled with task ID: %s", c.store.TaskID)
	return nil
}
