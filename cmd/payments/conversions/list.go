package conversions

import (
	"fmt"
	"math/big"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

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
	createdAtFromFlag    string
	createdAtToFlag      string
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
		fctl.WithShortDescription("List currency conversions ingested from exchange-style connectors"),
		fctl.WithStringFlag(c.connectorIDFlag, "", "Filter by connector ID"),
		fctl.WithStringFlag(c.referenceFlag, "", "Filter by PSP-assigned reference"),
		fctl.WithStringFlag(c.statusFlag, "", "Filter by status (PENDING|COMPLETED|FAILED|UNKNOWN)"),
		fctl.WithStringFlag(c.sourceAssetFlag, "", "Filter by source asset (e.g. USD/2)"),
		fctl.WithStringFlag(c.destinationAssetFlag, "", "Filter by destination asset (e.g. USDC/6)"),
		fctl.WithStringFlag(c.createdAtFromFlag, "", "Include only conversions created at or after this RFC3339 instant"),
		fctl.WithStringFlag(c.createdAtToFlag, "", "Include only conversions created at or before this RFC3339 instant"),
		fctl.WithCursorFlag(),
		fctl.WithPageSizeFlag(),
		fctl.WithController[*ListStore](c),
	)
}

// buildQuery translates the CLI filter flags into a V3QueryBuilder body
// (free-form map[string]any) using the same $match / $and pattern the
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
	addMatch("status", fctl.GetString(cmd, c.statusFlag))
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
	if !c.PaymentsVersion.IsAtLeast(versions.V3, conversionsMinMinor) {
		return nil, fmt.Errorf("conversions require Payments >= v3.%d (stack reports %s)", conversionsMinMinor, c.PaymentsVersion.Raw)
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
	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("conversions.list query=%v cursor=%q pageSize=%d", query, cursor, pageSize)
	_ = stackClient

	// TODO(EN-622): wire to stackClient.Payments.V3.ListConversions(cmd.Context(), operations.V3ListConversionsRequest{
	//     PageSize:       fctl.Ptr(int64(pageSize)),
	//     Cursor:         fctl.Ptr(cursor),
	//     V3QueryBuilder: query,
	// }) once formance-sdk-go/v3 exposes payments v3.3 endpoints (see EN-1012).
	// On success, map the response into c.store.Conversions (already aligned with V3Conversion)
	// and c.store.Cursor (use fctl.Cursor{HasMore, PageSize, Next, Previous}).
	return nil, fmt.Errorf("conversions.list: not wired yet - awaiting formance-sdk-go release with payments v3.3 (EN-622)")
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
