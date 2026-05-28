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

	// TODO(EN-618): wire to stackClient.Payments.V3.GetConversion(cmd.Context(), operations.V3GetConversionRequest{
	//     ConversionID: args[0],
	// }) once formance-sdk-go/v3 exposes payments v3.3 endpoints (see EN-1012).
	// On success, map the response Data into c.store.Conversion.
	return nil, fmt.Errorf("conversions.get: not wired yet - awaiting formance-sdk-go release with payments v3.3 (EN-618)")
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
