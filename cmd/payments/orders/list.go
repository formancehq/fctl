package orders

import (
	"fmt"
	"math/big"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	paymentsmodels "github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

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
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithController[*ListStore](c),
	)
}

// buildQuery turns the filter flags into a $match/$and query body. Keys are the
// payments storage column names (snake_case): the endpoint whitelists them and
// rejects anything else with a VALIDATION error.
func (c *ListController) buildQuery(cmd *cobra.Command) map[string]any {
	matches := make([]map[string]any, 0)
	addMatch := func(field, val string) {
		if val == "" {
			return
		}
		matches = append(matches, map[string]any{"$match": map[string]any{field: val}})
	}
	addMatch("connector_id", fctl.GetString(cmd, c.connectorIDFlag))
	addMatch("reference", fctl.GetString(cmd, c.referenceFlag))
	addMatch("direction", fctl.GetString(cmd, c.directionFlag))
	addMatch("status", fctl.GetString(cmd, c.statusFlag))
	addMatch("type", fctl.GetString(cmd, c.typeFlag))
	addMatch("source_asset", fctl.GetString(cmd, c.sourceAssetFlag))
	addMatch("destination_asset", fctl.GetString(cmd, c.destinationAssetFlag))

	if len(matches) == 0 {
		return nil
	}
	return map[string]any{"$and": matches}
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

	query := c.buildQuery(cmd)
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}
	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("orders.list query=%v cursor=%q pageSize=%d", query, cursor, pageSize)

	// The V3 query endpoints accept either a query body (first page) or an
	// opaque cursor (subsequent pages), never both.
	req := operations.V3ListOrdersRequest{}
	if cursor != "" {
		req.Cursor = fctl.Ptr(cursor)
	} else {
		req.RequestBody = query
		req.PageSize = fctl.Ptr(int64(pageSize))
	}

	res, err := stackClient.Payments.V3.ListOrders(cmd.Context(), req)
	if err != nil {
		return nil, err
	}
	if res.V3OrdersCursorResponse == nil {
		return nil, fmt.Errorf("orders.list: empty response (status %d)", res.StatusCode)
	}

	cur := res.V3OrdersCursorResponse.Cursor
	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("orders.list received=%d hasMore=%v", len(cur.Data), cur.HasMore)
	c.store.Orders = fctl.Map(cur.Data, toOrder)
	c.store.Cursor = fctl.Cursor{HasMore: cur.HasMore, PageSize: cur.PageSize, Next: cur.Next, Previous: cur.Previous}
	return c, nil
}

// toOrder maps an SDK V3Order onto the local store type, casting the typed
// string enums and reading metadata from the SDK's V3Metadata field.
func toOrder(o paymentsmodels.V3Order) Order {
	return Order{
		ID:                   o.ID,
		ConnectorID:          o.ConnectorID,
		Provider:             o.Provider,
		Reference:            o.Reference,
		ClientOrderID:        o.ClientOrderID,
		CreatedAt:            o.CreatedAt,
		UpdatedAt:            o.UpdatedAt,
		Direction:            string(o.V3OrderDirectionEnum),
		SourceAsset:          o.SourceAsset,
		DestinationAsset:     o.DestinationAsset,
		Type:                 string(o.V3OrderTypeEnum),
		Status:               string(o.V3OrderStatusEnum),
		BaseQuantityOrdered:  o.BaseQuantityOrdered,
		BaseQuantityFilled:   o.BaseQuantityFilled,
		LimitPrice:           o.LimitPrice,
		StopPrice:            o.StopPrice,
		TimeInForce:          string(o.V3TimeInForceEnum),
		ExpiresAt:            o.ExpiresAt,
		Fee:                  o.Fee,
		FeeAsset:             o.FeeAsset,
		AverageFillPrice:     o.AverageFillPrice,
		QuoteAmount:          o.QuoteAmount,
		QuoteAsset:           o.QuoteAsset,
		PriceAsset:           o.PriceAsset,
		SourceAccountID:      o.SourceAccountID,
		DestinationAccountID: o.DestinationAccountID,
		Metadata:             o.V3Metadata,
		Adjustments:          fctl.Map(o.Adjustments, toOrderAdjustment),
		Error:                o.Error,
	}
}

func toOrderAdjustment(a paymentsmodels.V3OrderAdjustment) OrderAdjustment {
	return OrderAdjustment{
		ID:                 a.ID,
		Reference:          a.Reference,
		CreatedAt:          a.CreatedAt,
		Status:             string(a.V3OrderStatusEnum),
		BaseQuantityFilled: a.BaseQuantityFilled,
		Fee:                a.Fee,
		FeeAsset:           a.FeeAsset,
		Metadata:           a.V3Metadata,
	}
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
