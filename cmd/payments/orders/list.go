package orders

import (
	"fmt"
	"math/big"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const ordersMinMinor = 3

type Order struct {
	ID                   string            `json:"id"`
	ConnectorID          string            `json:"connectorID"`
	Provider             string            `json:"provider"`
	Reference            string            `json:"reference"`
	ClientOrderID        *string           `json:"clientOrderID,omitempty"`
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
	Direction            string            `json:"direction"`
	SourceAsset          string            `json:"sourceAsset"`
	DestinationAsset     string            `json:"destinationAsset"`
	Type                 string            `json:"type"`
	Status               string            `json:"status"`
	BaseQuantityOrdered  *big.Int          `json:"baseQuantityOrdered"`
	BaseQuantityFilled   *big.Int          `json:"baseQuantityFilled,omitempty"`
	LimitPrice           *big.Int          `json:"limitPrice,omitempty"`
	StopPrice            *big.Int          `json:"stopPrice,omitempty"`
	TimeInForce          string            `json:"timeInForce"`
	ExpiresAt            *time.Time        `json:"expiresAt,omitempty"`
	Fee                  *big.Int          `json:"fee,omitempty"`
	FeeAsset             *string           `json:"feeAsset,omitempty"`
	AverageFillPrice     *big.Int          `json:"averageFillPrice,omitempty"`
	QuoteAmount          *big.Int          `json:"quoteAmount,omitempty"`
	QuoteAsset           *string           `json:"quoteAsset,omitempty"`
	PriceAsset           *string           `json:"priceAsset,omitempty"`
	SourceAccountID      *string           `json:"sourceAccountID,omitempty"`
	DestinationAccountID *string           `json:"destinationAccountID,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	Adjustments          []OrderAdjustment `json:"adjustments,omitempty"`
	Error                *string           `json:"error,omitempty"`
}

type OrderAdjustment struct {
	ID                 string            `json:"id"`
	Reference          string            `json:"reference"`
	CreatedAt          time.Time         `json:"createdAt"`
	Status             string            `json:"status"`
	BaseQuantityFilled *big.Int          `json:"baseQuantityFilled,omitempty"`
	Fee                *big.Int          `json:"fee,omitempty"`
	FeeAsset           *string           `json:"feeAsset,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	Raw                map[string]any    `json:"raw,omitempty"`
}

type ListStore struct {
	Orders []Order     `json:"orders"`
	Cursor fctl.Cursor `json:"cursor"`
}

type ListController struct {
	PaymentsVersion versions.Version
	store           *ListStore

	connectorIDFlag      string
	referenceFlag        string
	directionFlag        string
	statusFlag           string
	typeFlag             string
	sourceAssetFlag      string
	destinationAssetFlag string
	createdAtFromFlag    string
	createdAtToFlag      string
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func (c *ListController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

func NewListController() *ListController {
	return &ListController{
		store:                &ListStore{Orders: []Order{}},
		connectorIDFlag:      "connector-id",
		referenceFlag:        "reference",
		directionFlag:        "direction",
		statusFlag:           "status",
		typeFlag:             "type",
		sourceAssetFlag:      "source-asset",
		destinationAssetFlag: "destination-asset",
		createdAtFromFlag:    "created-at-from",
		createdAtToFlag:      "created-at-to",
	}
}

func (c *ListController) GetStore() *ListStore { return c.store }

func NewListCommand() *cobra.Command {
	c := NewListController()
	return fctl.NewCommand("list",
		fctl.WithAliases("ls", "l"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("List orders ingested from exchange-style connectors"),
		fctl.WithStringFlag(c.connectorIDFlag, "", "Filter by connector ID"),
		fctl.WithStringFlag(c.referenceFlag, "", "Filter by PSP-assigned reference"),
		fctl.WithStringFlag(c.directionFlag, "", "Filter by order direction (BUY|SELL|UNKNOWN)"),
		fctl.WithStringFlag(c.statusFlag, "", "Filter by status (PENDING|OPEN|PARTIALLY_FILLED|FILLED|CANCELLED|FAILED|EXPIRED|UNKNOWN)"),
		fctl.WithStringFlag(c.typeFlag, "", "Filter by order type (MARKET|LIMIT|STOP|STOP_LIMIT|TWAP|VWAP|PEG|BLOCK|RFQ|TRAILING_STOP|TRAILING_STOP_LIMIT|TAKE_PROFIT|TAKE_PROFIT_LIMIT|LIMIT_MAKER|UNKNOWN)"),
		fctl.WithStringFlag(c.sourceAssetFlag, "", "Filter by source asset (e.g. USD/2)"),
		fctl.WithStringFlag(c.destinationAssetFlag, "", "Filter by destination asset (e.g. BTC/8)"),
		fctl.WithStringFlag(c.createdAtFromFlag, "", "Include only orders created at or after this RFC3339 instant"),
		fctl.WithStringFlag(c.createdAtToFlag, "", "Include only orders created at or before this RFC3339 instant"),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithController[*ListStore](c),
	)
}

// buildQuery translates the CLI filter flags into a V3QueryBuilder body
// (free-form map[string]any) matching the $match / $and pattern that the
// ledger uses today (see cmd/ledger/accounts/list.go).
func (c *ListController) buildQuery(cmd *cobra.Command) (map[string]any, error) {
	matches := make([]map[string]any, 0)
	addMatch := func(field, val string) {
		if val == "" {
			return
		}
		matches = append(matches, map[string]any{"$match": map[string]any{field: val}})
	}
	addMatch("connectorID", fctl.GetString(cmd, c.connectorIDFlag))
	addMatch("reference", fctl.GetString(cmd, c.referenceFlag))
	addMatch("direction", fctl.GetString(cmd, c.directionFlag))
	addMatch("status", fctl.GetString(cmd, c.statusFlag))
	addMatch("type", fctl.GetString(cmd, c.typeFlag))
	addMatch("sourceAsset", fctl.GetString(cmd, c.sourceAssetFlag))
	addMatch("destinationAsset", fctl.GetString(cmd, c.destinationAssetFlag))

	from, err := fctl.GetDateTime(cmd, c.createdAtFromFlag)
	if err != nil {
		return nil, fmt.Errorf("parsing --%s: %w", c.createdAtFromFlag, err)
	}
	if from != nil {
		matches = append(matches, map[string]any{"$gte": map[string]any{"createdAt": from.Format(time.RFC3339Nano)}})
	}
	to, err := fctl.GetDateTime(cmd, c.createdAtToFlag)
	if err != nil {
		return nil, fmt.Errorf("parsing --%s: %w", c.createdAtToFlag, err)
	}
	if to != nil {
		matches = append(matches, map[string]any{"$lte": map[string]any{"createdAt": to.Format(time.RFC3339Nano)}})
	}

	if len(matches) == 0 {
		return nil, nil
	}
	return map[string]any{"$and": matches}, nil
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
	if err := versions.GetPaymentsVersion(cmd, nil, c); err != nil {
		return nil, err
	}
	if !c.PaymentsVersion.IsAtLeast(versions.V3, ordersMinMinor) {
		return nil, fmt.Errorf("orders require Payments >= v3.%d (stack reports %s)", ordersMinMinor, c.PaymentsVersion.Raw)
	}

	query, err := c.buildQuery(cmd)
	if err != nil {
		return nil, err
	}
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}
	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("orders.list query=%v cursor=%q pageSize=%d", query, cursor, pageSize)
	_ = stackClient

	// TODO(EN-1012): wire once fctl migrates to formance-sdk-go/v4. The payments
	// v3.3 endpoints shipped in v4.0.0 as a breaking major: pkg/models/components
	// was removed and the models were split into per-domain packages (e.g.
	// pkg/models/payments). Until that migration lands, this stub stays in place.
	//
	// Ready-to-paste wiring (replace this block and the return below):
	//
	//   import (
	//       operations "github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	//       paymentsmodels "github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"
	//   )
	//
	//   res, err := stackClient.Payments.V3.ListOrders(cmd.Context(), operations.V3ListOrdersRequest{
	//       RequestBody: query,                    // map[string]any, $and/$match
	//       Cursor:      fctl.Ptr(cursor),         // *string
	//       PageSize:    fctl.Ptr(int64(pageSize)),// *int64
	//   })
	//   if err != nil {
	//       return nil, err
	//   }
	//   cur := res.V3OrdersCursorResponse.Cursor
	//   c.store.Orders = fctl.Map(cur.Data, toOrder) // toOrder: paymentsmodels.V3Order -> Order
	//   c.store.Cursor = fctl.Cursor{
	//       HasMore:  cur.HasMore,
	//       PageSize: cur.PageSize, // already int64
	//       Next:     cur.Next,
	//       Previous: cur.Previous,
	//   }
	//   return c, nil
	//
	// Mapping notes (paymentsmodels.V3Order -> local Order):
	//   - typed-string enums; cast with:
	//       string(o.V3OrderDirectionEnum), string(o.V3OrderStatusEnum),
	//       string(o.V3OrderTypeEnum), string(o.V3TimeInForceEnum)
	//   - metadata field on V3Order is V3Metadata (not Metadata)
	//   - V3OrderAdjustment.Raw is *paymentsmodels.V3OrderAdjustmentRaw (empty
	//     struct in v4.0.0); leave OrderAdjustment.Raw nil or drop it from the
	//     store when wiring
	return nil, fmt.Errorf("orders.list: blocked until fctl migrates to formance-sdk-go/v4 (payments v3.3 shipped in v4.0.0 as a breaking major; see EN-1012)")
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	tableData := fctl.Map(c.store.Orders, func(o Order) []string {
		return []string{
			o.ID,
			o.Reference,
			o.Provider,
			o.Direction,
			o.Type,
			o.Status,
			o.SourceAsset,
			o.DestinationAsset,
			bigIntString(o.BaseQuantityOrdered),
			bigIntString(o.BaseQuantityFilled),
			o.CreatedAt.Format(time.RFC3339),
		}
	})
	tableData = fctl.Prepend(tableData, []string{
		"ID", "Reference", "Provider", "Direction", "Type", "Status",
		"SourceAsset", "DestinationAsset", "BaseQuantityOrdered", "BaseQuantityFilled", "CreatedAt",
	})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}
	return fctl.RenderCursor(cmd.OutOrStdout(), c.store.Cursor)
}

// bigIntString returns the decimal string of b, or "" when b is nil.
func bigIntString(b *big.Int) string {
	if b == nil {
		return ""
	}
	return b.String()
}

// strDeref returns the value of s, or "" when s is nil.
func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// timeRFC3339 formats t as RFC3339, or "" when t is nil.
func timeRFC3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
