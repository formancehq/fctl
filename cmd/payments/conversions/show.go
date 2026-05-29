package conversions

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
	Conversion *Conversion `json:"conversion"`
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
	return fctl.NewCommand("get <conversionID>",
		fctl.WithAliases("sh", "s"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("Get a single conversion by its Formance ID"),
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
	if !c.PaymentsVersion.IsAtLeast(versions.V3, conversionsMinMinor) {
		return nil, fmt.Errorf("conversions require Payments >= v3.%d (stack reports %s)", conversionsMinMinor, c.PaymentsVersion.Raw)
	}

	pterm.Debug.WithWriter(cmd.ErrOrStderr()).Printfln("conversions.show conversionID=%q", args[0])
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
	//   res, err := stackClient.Payments.V3.GetConversion(cmd.Context(), operations.V3GetConversionRequest{
	//       ConversionID: args[0],
	//   })
	//   if err != nil {
	//       return nil, err
	//   }
	//   c.store.Conversion = toConversion(res.V3GetConversionResponse.V3Conversion) // .V3Conversion is the data payload
	//   return c, nil
	//
	// Mapping notes (paymentsmodels.V3Conversion -> local Conversion): same
	// caveats as conversions.list — cast string(cv.V3ConversionStatusEnum) and
	// read the metadata from V3Metadata (not Metadata).
	return nil, fmt.Errorf("conversions.get: blocked until fctl migrates to formance-sdk-go/v4 (payments v3.3 shipped in v4.0.0 as a breaking major; see EN-1012)")
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	if c.store.Conversion == nil {
		fctl.Println("No conversion data.")
		return nil
	}
	cv := c.store.Conversion
	out := cmd.OutOrStdout()

	fctl.Section.WithWriter(out).Println("Information")
	info := pterm.TableData{
		{pterm.LightCyan("ID"), cv.ID},
		{pterm.LightCyan("Reference"), cv.Reference},
		{pterm.LightCyan("ConnectorID"), cv.ConnectorID},
		{pterm.LightCyan("Provider"), cv.Provider},
		{pterm.LightCyan("Status"), cv.Status},
		{pterm.LightCyan("SourceAsset"), cv.SourceAsset},
		{pterm.LightCyan("DestinationAsset"), cv.DestinationAsset},
		{pterm.LightCyan("SourceAmount"), bigIntString(cv.SourceAmount)},
		{pterm.LightCyan("DestinationAmount"), bigIntString(cv.DestinationAmount)},
		{pterm.LightCyan("Fee"), bigIntString(cv.Fee)},
		{pterm.LightCyan("FeeAsset"), strDeref(cv.FeeAsset)},
		{pterm.LightCyan("SourceAccountID"), strDeref(cv.SourceAccountID)},
		{pterm.LightCyan("DestinationAccountID"), strDeref(cv.DestinationAccountID)},
		{pterm.LightCyan("CreatedAt"), cv.CreatedAt.Format(time.RFC3339)},
		{pterm.LightCyan("UpdatedAt"), cv.UpdatedAt.Format(time.RFC3339)},
		{pterm.LightCyan("Error"), strDeref(cv.Error)},
	}
	if err := pterm.DefaultTable.WithWriter(out).WithData(info).Render(); err != nil {
		return err
	}

	return fctl.PrintMetadata(out, metadata.Metadata(cv.Metadata))
}
