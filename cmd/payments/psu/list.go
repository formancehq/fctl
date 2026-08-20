package psu

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	PaymentServiceUsers []payments.V3PaymentServiceUser `json:"paymentServiceUsers"`
	Cursor              fctl.Cursor                     `json:"cursor"`
}

type ListController struct {
	PaymentsVersion versions.Version
	store           *ListStore
}

func (c *ListController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListController() *ListController {
	return &ListController{
		store: &ListStore{},
	}
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("List payment service users"),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithController[*ListStore](c),
	)
}

func (c *ListController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}

	// The V3 query endpoints accept either a query body (first page) or an
	// opaque cursor (subsequent pages), never both.
	req := operations.V3ListPaymentServiceUsersRequest{}
	if cursor != "" {
		req.Cursor = fctl.Ptr(cursor)
	} else {
		req.PageSize = fctl.Ptr(int64(pageSize))
	}

	response, err := stackClient.Payments.V3.ListPaymentServiceUsers(cmd.Context(), req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	if response.V3PaymentServiceUsersCursorResponse == nil {
		return nil, fmt.Errorf("unexpected empty response")
	}

	cur := response.V3PaymentServiceUsersCursorResponse.Cursor
	c.store.PaymentServiceUsers = cur.Data
	c.store.Cursor = fctl.Cursor{HasMore: cur.HasMore, PageSize: cur.PageSize, Next: cur.Next, Previous: cur.Previous}

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	tableData := fctl.Map(c.store.PaymentServiceUsers, func(u payments.V3PaymentServiceUser) []string {
		return []string{
			u.ID,
			u.Name,
			u.CreatedAt.Format(time.RFC3339),
			fmt.Sprintf("%d", len(u.BankAccountIDs)),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "Name", "CreatedAt", "BankAccounts"})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return fctl.RenderCursor(cmd.OutOrStdout(), c.store.Cursor)
}
