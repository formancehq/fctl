package psu

import (
	"errors"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ForwardStore struct {
	PaymentServiceUserID string `json:"paymentServiceUserID"`
	ConnectorID          string `json:"connectorID"`
}

type ForwardController struct {
	PaymentsVersion versions.Version
	store           *ForwardStore
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
	return fctl.NewCommand("forward <paymentServiceUserID> <connectorID>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Associate a payment service user with a connector (required before create-link)"),
		fctl.WithAliases("fo", "f"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*ForwardStore](c),
	)
}

func (c *ForwardController) GetStore() *ForwardStore {
	return c.store
}

func (c *ForwardController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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

	paymentServiceUserID := args[0]
	if paymentServiceUserID == "" {
		return nil, errors.New("payment service user ID is required")
	}

	connectorID := args[1]
	if connectorID == "" {
		return nil, errors.New("connector ID is required")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to associate payment service user '%s' with connector '%s'", paymentServiceUserID, connectorID) {
		return nil, fctl.ErrMissingApproval
	}

	response, err := stackClient.Payments.V3.ForwardPaymentServiceUserToProvider(cmd.Context(), operations.V3ForwardPaymentServiceUserToProviderRequest{
		PaymentServiceUserID: paymentServiceUserID,
		ConnectorID:          connectorID,
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.PaymentServiceUserID = paymentServiceUserID
	c.store.ConnectorID = connectorID

	return c, nil
}

func (c *ForwardController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Payment service user '%s' forwarded to connector '%s'.", c.store.PaymentServiceUserID, c.store.ConnectorID)
	return nil
}
