package psu

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"
	"github.com/formancehq/go-libs/v4/metadata"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	PaymentServiceUser *payments.V3PaymentServiceUser `json:"paymentServiceUser"`
}

type ShowController struct {
	PaymentsVersion versions.Version
	store           *ShowStore
}

func (c *ShowController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("get <paymentServiceUserID>",
		fctl.WithShortDescription("Get a payment service user"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithAliases("sh", "s", "show"),
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

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion.Major < versions.V3 {
		return nil, fmt.Errorf("payment service users require Payments API v3")
	}

	response, err := stackClient.Payments.V3.GetPaymentServiceUser(cmd.Context(), operations.V3GetPaymentServiceUserRequest{
		PaymentServiceUserID: args[0],
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	if response.V3GetPaymentServiceUserResponse == nil {
		return nil, fmt.Errorf("unexpected empty response")
	}

	psu := response.V3GetPaymentServiceUserResponse.V3PaymentServiceUser
	c.store.PaymentServiceUser = &psu

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Information")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.PaymentServiceUser.ID})
	tableData = append(tableData, []string{pterm.LightCyan("Name"), c.store.PaymentServiceUser.Name})
	tableData = append(tableData, []string{pterm.LightCyan("CreatedAt"), c.store.PaymentServiceUser.CreatedAt.Format(time.RFC3339)})
	tableData = append(tableData, []string{pterm.LightCyan("BankAccountIDs"), fmt.Sprintf("%v", c.store.PaymentServiceUser.BankAccountIDs)})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return fctl.PrintMetadata(cmd.OutOrStdout(), metadata.Metadata(c.store.PaymentServiceUser.V3Metadata))
}
