package bankaccounts

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ForwardStore struct {
	BankAccountID string `json:"bankAccountId"`
	ConnectorID   string `json:"connectorId"`

	// V3
	TaskID string `json:"taskId"`
}
type ForwardController struct {
	PaymentsVersion versions.Version

	store *ForwardStore
}

func (c *ForwardController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ForwardStore] = (*ForwardController)(nil)

func NewForwardStore() *ForwardStore {
	return &ForwardStore{}
}

func NewForwardController() *ForwardController {
	return &ForwardController{
		store: NewForwardStore(),
	}
}

func NewForwardCommand() *cobra.Command {
	c := NewForwardController()
	return fctl.NewCommand("forward <bankAccountID> <connectorID>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Forward a bank account to a connector"),
		fctl.WithAliases("fo", "f"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithController[*ForwardStore](c),
	)
}

func (c *ForwardController) GetStore() *ForwardStore {
	return c.store
}

func (c *ForwardController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V1 {
		return nil, fmt.Errorf("bank accounts are only supported in >= v1.0.0")
	}

	if !fctl.CheckStackApprobation(cmd, store.Stack(), "You are about to forward a bank account to a connector") {
		return nil, fctl.ErrMissingApproval
	}

	bankAccountID := args[0]
	if bankAccountID == "" {
		return nil, errors.New("bank account ID is required")
	}

	connectorID := args[1]
	if connectorID == "" {
		return nil, errors.New("connector ID is required")
	}

	if c.PaymentsVersion < versions.V3 {
		//nolint:gosimple
		response, err := store.Client().Payments.V1.ForwardBankAccount(cmd.Context(), operations.ForwardBankAccountRequest{
			ForwardBankAccountRequest: shared.ForwardBankAccountRequest{
				ConnectorID: connectorID,
			},
			BankAccountID: bankAccountID,
		})
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		c.store.BankAccountID = response.BankAccountResponse.Data.ID
		c.store.ConnectorID = *response.BankAccountResponse.Data.ConnectorID

		return c, nil
	}

	response, err := store.Client().Payments.V3.ForwardBankAccount(cmd.Context(), operations.V3ForwardBankAccountRequest{
		V3ForwardBankAccountRequest: &shared.V3ForwardBankAccountRequest{
			ConnectorID: connectorID,
		},
		BankAccountID: bankAccountID,
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.TaskID = response.V3ForwardBankAccountResponse.Data.GetTaskID()
	return c, nil
}

func (c *ForwardController) Render(cmd *cobra.Command, args []string) error {
	if c.PaymentsVersion < versions.V3 {
		pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Bank Account %s forwarded to connector %s", c.store.BankAccountID, c.store.ConnectorID)
		return nil
	}
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Forwarding Bank Account scheduled with TaskID: %s", c.store.TaskID)
	return nil
}
