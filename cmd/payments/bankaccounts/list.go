package bankaccounts

import (
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ListStore struct {
	Cursor *shared.V3BankAccountsCursorResponseCursor `json:"cursor"`
}

type ListController struct {
	PaymentsVersion versions.Version

	store *ListStore

	cursorFlag   string
	pageSizeFlag string
}

func (c *ListController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListStore() *ListStore {
	return &ListStore{
		Cursor: &shared.V3BankAccountsCursorResponseCursor{},
	}
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
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V1 {
		return nil, fmt.Errorf("bank accounts are only supported in >= v1.0.0")
	}

	var cursor *string
	if c := fctl.GetString(cmd, c.cursorFlag); c != "" {
		cursor = &c
	}

	var pageSize *int64
	if ps := fctl.GetInt(cmd, c.pageSizeFlag); ps > 0 {
		pageSize = fctl.Ptr(int64(ps))
	}

	if c.PaymentsVersion >= versions.V3 {
		return c.v3list(cmd, stackClient, cursor, pageSize)
	}

	response, err := stackClient.Payments.V1.ListBankAccounts(
		cmd.Context(),
		operations.ListBankAccountsRequest{
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

	c.store.Cursor = ToV3BankAccountCursor(&response.BankAccountsCursor.Cursor)

	return c, nil
}

func (c *ListController) v3list(cmd *cobra.Command, stackClient *formance.Formance, cursor *string, pageSize *int64) (fctl.Renderable, error) {
	response, err := stackClient.Payments.V3.ListBankAccounts(
		cmd.Context(),
		operations.V3ListBankAccountsRequest{
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

	c.store.Cursor = &response.V3BankAccountsCursorResponse.Cursor

	return c, nil
}

func ToV3BankAccountCursor(c *shared.BankAccountsCursorCursor) *shared.V3BankAccountsCursorResponseCursor {
	cursor := &shared.V3BankAccountsCursorResponseCursor{
		Data:     make([]shared.V3BankAccount, 0, len(c.Data)),
		HasMore:  c.HasMore,
		Next:     c.Next,
		Previous: c.Previous,
		PageSize: c.PageSize,
	}
	for _, acc := range c.Data {
		cursor.Data = append(cursor.Data, ToV3BankAccount(&acc))
	}
	return cursor
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	tableData := fctl.Map(c.store.Cursor.Data, func(bc shared.V3BankAccount) []string {
		row := []string{
			bc.ID,
			bc.Name,
			bc.CreatedAt.Format(time.RFC3339),
			"",
		}
		if bc.Country != nil {
			row[3] = *bc.Country
		}
		return row
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "Name", "CreatedAt", "Country"})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Paging")
	cursorTable := pterm.TableData{}
	cursorTable = append(cursorTable, []string{pterm.LightCyan("Page Size"), fmt.Sprintf("%d", c.store.Cursor.PageSize)})
	cursorTable = append(cursorTable, []string{pterm.LightCyan("Has More"), fmt.Sprintf("%t", c.store.Cursor.HasMore)})
	if c.store.Cursor.Next != nil {
		cursorTable = append(cursorTable, []string{pterm.LightCyan("Next"), *c.store.Cursor.Next})
	}
	if c.store.Cursor.Previous != nil {
		cursorTable = append(cursorTable, []string{pterm.LightCyan("Previous"), *c.store.Cursor.Previous})
	}
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(cursorTable).
		Render(); err != nil {
		return err
	}

	return nil
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithShortDescription("List bank accounts"),
		fctl.WithStringFlag(c.cursorFlag, "", "Cursor"),
		fctl.WithIntFlag(c.pageSizeFlag, 0, "PageSize"),
		fctl.WithController[*ListStore](c),
	)
}
