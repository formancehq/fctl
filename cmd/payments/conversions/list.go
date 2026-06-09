package conversions

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

const conversionsMinMinor = 3

type Conversion struct {
	ID                   string            `json:"id"`
	ConnectorID          string            `json:"connectorID"`
	Provider             string            `json:"provider"`
	Reference            string            `json:"reference"`
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
	SourceAsset          string            `json:"sourceAsset"`
	DestinationAsset     string            `json:"destinationAsset"`
	SourceAmount         *big.Int          `json:"sourceAmount"`
	DestinationAmount    *big.Int          `json:"destinationAmount,omitempty"`
	Fee                  *big.Int          `json:"fee,omitempty"`
	FeeAsset             *string           `json:"feeAsset,omitempty"`
	Status               string            `json:"status"`
	SourceAccountID      *string           `json:"sourceAccountID,omitempty"`
	DestinationAccountID *string           `json:"destinationAccountID,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	Error                *string           `json:"error,omitempty"`
}

type ListStore struct {
	Conversions []Conversion `json:"conversions"`
	Cursor      fctl.Cursor  `json:"cursor"`
}

type ListController struct {
	PaymentsVersion versions.Version
	store           *ListStore

	connectorIDFlag      string
	referenceFlag        string
	statusFlag           string
	sourceAssetFlag      string
	destinationAssetFlag string
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func (c *ListController) SetVersion(version versions.Version) { c.PaymentsVersion = version }

func NewListController() *ListController {
	return &ListController{
		store:                &ListStore{Conversions: []Conversion{}},
		connectorIDFlag:      "connector-id",
		referenceFlag:        "reference",
		statusFlag:           "status",
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
		fctl.WithShortDescription("List currency conversions ingested from exchange-style connectors"),
		fctl.WithStringFlag(c.connectorIDFlag, "", "Filter by connector ID"),
		fctl.WithStringFlag(c.referenceFlag, "", "Filter by PSP-assigned reference"),
		fctl.WithStringFlag(c.statusFlag, "", "Filter by status (PENDING|COMPLETED|FAILED|UNKNOWN)"),
		fctl.WithStringFlag(c.sourceAssetFlag, "", "Filter by source asset (e.g. USD/2)"),
		fctl.WithStringFlag(c.destinationAssetFlag, "", "Filter by destination asset (e.g. USDC/6)"),
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
	addMatch("status", fctl.GetString(cmd, c.statusFlag))
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
	if !c.PaymentsVersion.IsAtLeast(versions.V3, conversionsMinMinor) {
		return nil, fmt.Errorf("conversions require Payments >= v3.%d (stack reports %s)", conversionsMinMinor, c.PaymentsVersion.Raw)
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
	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("conversions.list query=%v cursor=%q pageSize=%d", query, cursor, pageSize)

	// The V3 query endpoints accept either a query body (first page) or an
	// opaque cursor (subsequent pages), never both.
	req := operations.V3ListConversionsRequest{}
	if cursor != "" {
		req.Cursor = fctl.Ptr(cursor)
	} else {
		req.RequestBody = query
		req.PageSize = fctl.Ptr(int64(pageSize))
	}

	res, err := stackClient.Payments.V3.ListConversions(cmd.Context(), req)
	if err != nil {
		return nil, err
	}
	if res.V3ConversionsCursorResponse == nil {
		return nil, fmt.Errorf("conversions.list: empty response (status %d)", res.StatusCode)
	}

	cur := res.V3ConversionsCursorResponse.Cursor
	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("conversions.list received=%d hasMore=%v", len(cur.Data), cur.HasMore)
	c.store.Conversions = fctl.Map(cur.Data, toConversion)
	c.store.Cursor = fctl.Cursor{HasMore: cur.HasMore, PageSize: cur.PageSize, Next: cur.Next, Previous: cur.Previous}
	return c, nil
}

// toConversion maps an SDK V3Conversion onto the local nil-safe store type.
// The status enum is a typed string and metadata lives under V3Metadata.
func toConversion(cv paymentsmodels.V3Conversion) Conversion {
	return Conversion{
		ID:                   cv.ID,
		ConnectorID:          cv.ConnectorID,
		Provider:             cv.Provider,
		Reference:            cv.Reference,
		CreatedAt:            cv.CreatedAt,
		UpdatedAt:            cv.UpdatedAt,
		SourceAsset:          cv.SourceAsset,
		DestinationAsset:     cv.DestinationAsset,
		SourceAmount:         cv.SourceAmount,
		DestinationAmount:    cv.DestinationAmount,
		Fee:                  cv.Fee,
		FeeAsset:             cv.FeeAsset,
		Status:               string(cv.V3ConversionStatusEnum),
		SourceAccountID:      cv.SourceAccountID,
		DestinationAccountID: cv.DestinationAccountID,
		Metadata:             cv.V3Metadata,
		Error:                cv.Error,
	}
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	tableData := fctl.Map(c.store.Conversions, func(cv Conversion) []string {
		return []string{
			cv.ID,
			cv.Reference,
			cv.Provider,
			cv.Status,
			cv.SourceAsset,
			cv.DestinationAsset,
			bigIntString(cv.SourceAmount),
			bigIntString(cv.DestinationAmount),
			cv.CreatedAt.Format(time.RFC3339),
		}
	})
	tableData = fctl.Prepend(tableData, []string{
		"ID", "Reference", "Provider", "Status",
		"SourceAsset", "DestinationAsset", "SourceAmount", "DestinationAmount", "CreatedAt",
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
