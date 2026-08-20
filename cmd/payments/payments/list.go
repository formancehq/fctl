package payments

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	paymentsmodels "github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	// V1
	Cursor *paymentsmodels.PaymentsCursorCursorBase `json:"cursor,omitempty"`
	// V3
	V3Cursor *paymentsmodels.V3PaymentsCursorResponseCursor `json:"v3Cursor,omitempty"`
}
type ListController struct {
	PaymentsVersion versions.Version
	store           *ListStore

	cursorFlag   string
	pageSizeFlag string
}

func (c *ListController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store: NewListStore(),

		cursorFlag:   "cursor",
		pageSizeFlag: "page-size",
	}
}

func (c *ListController) GetStore() *ListStore {
	return c.store
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

	var cursor *string
	if c := fctl.GetString(cmd, c.cursorFlag); c != "" {
		cursor = &c
	}

	var pageSize *int64
	if ps := fctl.GetInt(cmd, c.pageSizeFlag); ps > 0 {
		pageSize = fctl.Ptr(int64(ps))
	}

	// The V1 Payment.Connector field is a closed enum generated from an older
	// spec: it doesn't know about connectors added since (e.g. plaid), and hard
	// fails unmarshaling any payment routed through one of them. V3's Provider
	// field is a plain string, so it doesn't have this problem.
	if c.PaymentsVersion.Major >= versions.V3 {
		response, err := stackClient.Payments.V3.ListPayments(
			cmd.Context(),
			operations.V3ListPaymentsRequest{
				Cursor:   cursor,
				PageSize: pageSize,
			},
		)
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
		if response.V3PaymentsCursorResponse == nil {
			return nil, fmt.Errorf("unexpected empty response")
		}

		c.store.V3Cursor = &response.V3PaymentsCursorResponse.Cursor
		return c, nil
	}

	response, err := stackClient.Payments.V1.ListPayments(
		cmd.Context(),
		operations.ListPaymentsRequest{
			Cursor:   cursor,
			PageSize: pageSize,
		},
	)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Cursor = &response.PaymentsCursor.CursorBase

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	if c.PaymentsVersion.Major >= versions.V3 {
		return c.renderV3(cmd)
	}
	return c.renderV1(cmd)
}

func (c *ListController) renderV1(cmd *cobra.Command) error {
	tableData := fctl.Map(c.store.Cursor.Data, func(payment paymentsmodels.Payment) []string {
		return []string{
			payment.ID,
			string(payment.PaymentType),
			fmt.Sprint(payment.Amount),
			fmt.Sprint(payment.InitialAmount),
			payment.Asset,
			string(payment.PaymentStatus),
			string(payment.PaymentScheme),
			payment.Reference,
			payment.SourceAccountID,
			payment.DestinationAccountID,
			payment.ConnectorID,
			payment.CreatedAt.Format(time.RFC3339),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "Type", "Amount", "InitialAmount", "Asset", "Status",
		"Scheme", "Reference", "Source Account ID", "Destination Account ID", "ConnectorID", "CreatedAt"})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	tableData = pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("HasMore"), fmt.Sprintf("%v", c.store.Cursor.HasMore)})
	tableData = append(tableData, []string{pterm.LightCyan("PageSize"), fmt.Sprintf("%d", c.store.Cursor.PageSize)})
	tableData = append(tableData, []string{pterm.LightCyan("Next"), func() string {
		if c.store.Cursor.Next == nil {
			return ""
		}
		return *c.store.Cursor.Next
	}()})
	tableData = append(tableData, []string{pterm.LightCyan("Previous"), func() string {
		if c.store.Cursor.Previous == nil {
			return ""
		}
		return *c.store.Cursor.Previous
	}()})

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}

func (c *ListController) renderV3(cmd *cobra.Command) error {
	tableData := fctl.Map(c.store.V3Cursor.Data, func(payment paymentsmodels.V3Payment) []string {
		return []string{
			payment.ID,
			payment.Provider,
			string(payment.V3PaymentTypeEnum),
			fmt.Sprint(payment.Amount),
			fmt.Sprint(payment.InitialAmount),
			payment.Asset,
			string(payment.V3PaymentStatusEnum),
			payment.Scheme,
			payment.Reference,
			payment.ConnectorID,
			payment.CreatedAt.Format(time.RFC3339),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "Provider", "Type", "Amount", "InitialAmount", "Asset",
		"Status", "Scheme", "Reference", "ConnectorID", "CreatedAt"})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return fctl.RenderCursor(cmd.OutOrStdout(), fctl.Cursor{
		HasMore:  c.store.V3Cursor.HasMore,
		PageSize: c.store.V3Cursor.PageSize,
		Next:     c.store.V3Cursor.Next,
		Previous: c.store.V3Cursor.Previous,
	})
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("List payments"),
		fctl.WithStringFlag(c.cursorFlag, "", "Pagination cursor"),
		fctl.WithIntFlag(c.pageSizeFlag, 0, "Page size"),
		fctl.WithController[*ListStore](c),
	)
}
