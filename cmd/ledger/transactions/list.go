package transactions

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/collectionutils"
	"github.com/formancehq/go-libs/v3/pointer"

	internal "github.com/formancehq/fctl/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/pkg"
)

type ListStore struct {
	Transaction shared.TransactionsCursorResponseCursor `json:"transactionCursor"`
}
type ListController struct {
	store           *ListStore
	pageSizeFlag    string
	metadataFlag    string
	referenceFlag   string
	accountFlag     string
	destinationFlag string
	sourceFlag      string
	endTimeFlag     string
	startTimeFlag   string
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewDefaultListStore() *ListStore {
	return &ListStore{}
}

func NewListController() *ListController {
	return &ListController{
		store:           NewDefaultListStore(),
		pageSizeFlag:    "page-size",
		metadataFlag:    "metadata",
		referenceFlag:   "reference",
		accountFlag:     "account",
		destinationFlag: "dst",
		sourceFlag:      "src",
		endTimeFlag:     "end",
		startTimeFlag:   "start",
	}
}

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List transactions"),
		fctl.WithStringFlag(c.accountFlag, "", "Filter on account"),
		fctl.WithStringFlag(c.destinationFlag, "", "Filter on destination account"),
		fctl.WithStringFlag(c.endTimeFlag, "", "Consider transactions before date"),
		fctl.WithStringFlag(c.startTimeFlag, "", "Consider transactions after date"),
		fctl.WithStringFlag(c.sourceFlag, "", "Filter on source account"),
		fctl.WithStringFlag(c.referenceFlag, "", "Filter on reference"),
		fctl.WithStringSliceFlag(c.metadataFlag, []string{}, "Filter transactions with metadata"),
		fctl.WithIntFlag(c.pageSizeFlag, 5, "Page size"),
		fctl.WithHiddenFlag(c.metadataFlag),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithController[*ListStore](c),
	)
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	metadata, err := fctl.ParseMetadata(fctl.GetStringSlice(cmd, c.metadataFlag))
	if err != nil {
		return nil, err
	}

	ledger := fctl.GetString(cmd, internal.LedgerFlag)
	req := operations.ListTransactionsRequest{
		Ledger:   ledger,
		PageSize: fctl.Ptr(int64(fctl.GetInt(cmd, c.pageSizeFlag))),
	}

	if account := fctl.GetString(cmd, c.accountFlag); account != "" {
		req.Account = pointer.For(account)
	}
	if source := fctl.GetString(cmd, c.sourceFlag); source != "" {
		req.Source = pointer.For(source)
	}
	if destination := fctl.GetString(cmd, c.destinationFlag); destination != "" {
		req.Destination = pointer.For(destination)
	}
	if reference := fctl.GetString(cmd, c.referenceFlag); reference != "" {
		req.Reference = pointer.For(reference)
	}
	if startTime := fctl.GetString(cmd, c.startTimeFlag); startTime != "" {
		t, err := time.Parse(time.RFC3339Nano, startTime)
		if err != nil {
			return nil, fmt.Errorf("parsing start time: %w", err)
		}
		req.StartTime = pointer.For(t)
	}
	if endTime := fctl.GetString(cmd, c.endTimeFlag); endTime != "" {
		t, err := time.Parse(time.RFC3339Nano, endTime)
		if err != nil {
			return nil, fmt.Errorf("parsing end time: %w", err)
		}
		req.EndTime = pointer.For(t)
	}
	req.Metadata = collectionutils.ConvertMap(metadata, collectionutils.ToAny[string])

	response, err := stackClient.Ledger.V1.ListTransactions(cmd.Context(), req)
	if err != nil {
		return nil, err
	}

	c.store.Transaction = response.TransactionsCursorResponse.Cursor

	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, args []string) error {
	if len(c.store.Transaction.Data) == 0 {
		fctl.Println("No transactions found.")
		return nil
	}

	tableData := fctl.Map(c.store.Transaction.Data, func(tx shared.Transaction) []string {
		return []string{
			fmt.Sprintf("%d", tx.Txid),
			func() string {
				if tx.Reference == nil {
					return ""
				}
				return *tx.Reference
			}(),
			tx.Timestamp.Format(time.RFC3339),
			fctl.MetadataAsShortString(tx.Metadata),
		}
	})
	tableData = fctl.Prepend(tableData, []string{"ID", "Reference", "Date", "Metadata"})

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
