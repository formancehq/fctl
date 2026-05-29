package orders

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v4/metadata"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	Order *Order `json:"order"`
}

type ShowController struct {
	PaymentsVersion versions.Version
	store           *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func (c *ShowController) SetVersion(version versions.Version) { c.PaymentsVersion = version }

func NewShowController() *ShowController {
	return &ShowController{store: &ShowStore{}}
}

func (c *ShowController) GetStore() *ShowStore { return c.store }

func NewShowCommand() *cobra.Command {
	c := NewShowController()
	return fctl.NewCommand("get <orderID>",
		fctl.WithAliases("sh", "s"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("Get a single order by its Formance ID, including its full adjustments history"),
		fctl.WithController[*ShowStore](c),
	)
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
	if !c.PaymentsVersion.IsAtLeast(versions.V3, ordersMinMinor) {
		return nil, fmt.Errorf("orders require Payments >= v3.%d (stack reports %s)", ordersMinMinor, c.PaymentsVersion.Raw)
	}

	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("orders.show orderID=%q", args[0])
	_ = stackClient

	// TODO(EN-1012): wire once fctl migrates to formance-sdk-go/v4. The payments
	// v3.3 endpoints shipped in v4.0.0 as a breaking major (pkg/models/components
	// removed, models split into per-domain packages). Until that migration
	// lands, this stub stays in place.
	//
	// Ready-to-paste wiring (replace this block and the return below):
	//
	//   import (
	//       operations "github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	//       paymentsmodels "github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"
	//   )
	//
	//   res, err := stackClient.Payments.V3.GetOrder(cmd.Context(), operations.V3GetOrderRequest{
	//       OrderID: args[0],
	//   })
	//   if err != nil {
	//       return nil, err
	//   }
	//   c.store.Order = toOrder(res.V3GetOrderResponse.V3Order) // .V3Order is the data payload
	//   return c, nil
	//
	// Mapping notes (paymentsmodels.V3Order -> local Order): same caveats as
	// orders.list — cast typed-string enums (V3OrderDirectionEnum, etc.), use
	// V3Metadata (not Metadata), and V3OrderAdjustment.Raw is an empty struct
	// in v4.0.0 (leave OrderAdjustment.Raw nil or drop it).
	return nil, fmt.Errorf("orders.get: blocked until fctl migrates to formance-sdk-go/v4 (payments v3.3 shipped in v4.0.0 as a breaking major; see EN-1012)")
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	if c.store.Order == nil {
		fctl.Println("No order data.")
		return nil
	}
	o := c.store.Order
	out := cmd.OutOrStdout()

	fctl.Section.WithWriter(out).Println("Information")
	info := pterm.TableData{
		{pterm.LightCyan("ID"), o.ID},
		{pterm.LightCyan("Reference"), o.Reference},
		{pterm.LightCyan("ConnectorID"), o.ConnectorID},
		{pterm.LightCyan("Provider"), o.Provider},
		{pterm.LightCyan("ClientOrderID"), strDeref(o.ClientOrderID)},
		{pterm.LightCyan("Direction"), o.Direction},
		{pterm.LightCyan("Type"), o.Type},
		{pterm.LightCyan("Status"), o.Status},
		{pterm.LightCyan("TimeInForce"), o.TimeInForce},
		{pterm.LightCyan("SourceAsset"), o.SourceAsset},
		{pterm.LightCyan("DestinationAsset"), o.DestinationAsset},
		{pterm.LightCyan("BaseQuantityOrdered"), bigIntString(o.BaseQuantityOrdered)},
		{pterm.LightCyan("BaseQuantityFilled"), bigIntString(o.BaseQuantityFilled)},
		{pterm.LightCyan("LimitPrice"), bigIntString(o.LimitPrice)},
		{pterm.LightCyan("StopPrice"), bigIntString(o.StopPrice)},
		{pterm.LightCyan("AverageFillPrice"), bigIntString(o.AverageFillPrice)},
		{pterm.LightCyan("QuoteAmount"), bigIntString(o.QuoteAmount)},
		{pterm.LightCyan("QuoteAsset"), strDeref(o.QuoteAsset)},
		{pterm.LightCyan("PriceAsset"), strDeref(o.PriceAsset)},
		{pterm.LightCyan("Fee"), bigIntString(o.Fee)},
		{pterm.LightCyan("FeeAsset"), strDeref(o.FeeAsset)},
		{pterm.LightCyan("SourceAccountID"), strDeref(o.SourceAccountID)},
		{pterm.LightCyan("DestinationAccountID"), strDeref(o.DestinationAccountID)},
		{pterm.LightCyan("ExpiresAt"), timeRFC3339(o.ExpiresAt)},
		{pterm.LightCyan("CreatedAt"), o.CreatedAt.Format(time.RFC3339)},
		{pterm.LightCyan("UpdatedAt"), o.UpdatedAt.Format(time.RFC3339)},
		{pterm.LightCyan("Error"), strDeref(o.Error)},
	}
	if err := pterm.DefaultTable.WithWriter(out).WithData(info).Render(); err != nil {
		return err
	}

	if err := fctl.PrintMetadata(out, metadata.Metadata(o.Metadata)); err != nil {
		return err
	}

	if len(o.Adjustments) == 0 {
		return nil
	}
	fctl.Section.WithWriter(out).Println("Adjustments")
	adj := fctl.Map(o.Adjustments, func(a OrderAdjustment) []string {
		return []string{
			a.CreatedAt.Format(time.RFC3339),
			a.Status,
			bigIntString(a.BaseQuantityFilled),
			bigIntString(a.Fee),
			strDeref(a.FeeAsset),
			a.ID,
		}
	})
	adj = fctl.Prepend(adj, []string{"CreatedAt", "Status", "BaseQuantityFilled", "Fee", "FeeAsset", "ID"})
	return pterm.DefaultTable.WithHasHeader().WithWriter(out).WithData(adj).Render()
}
