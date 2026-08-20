package psu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateStore struct {
	PaymentServiceUserID string `json:"paymentServiceUserID"`
}

type CreateController struct {
	PaymentsVersion versions.Version

	store *CreateStore
}

func (c *CreateController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	c := NewCreateController()
	return fctl.NewCommand("create <file>|-",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Create a payment service user"),
		fctl.WithAliases("cr", "c"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*CreateStore](c),
	)
}

func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion.Major < versions.V3 {
		return nil, fmt.Errorf("payment service users require Payments API v3")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to create a payment service user") {
		return nil, fctl.ErrMissingApproval
	}

	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	// Sent as a raw JSON body (rather than the generated SDK request struct) so
	// that fields the SDK hasn't caught up with yet - e.g. contactDetails.locale,
	// required by some open banking providers such as Plaid - still reach the API.
	var body map[string]any
	if err := json.Unmarshal([]byte(script), &body); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	url := strings.TrimRight(clients.URI, "/") + "/api/payments/v3/payment-service-users"
	req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	httpResp, err := clients.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("creating payment service user: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d: %s", httpResp.StatusCode, string(respBody))
	}

	var result struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	c.store.PaymentServiceUserID = result.Data

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Payment service user created with ID: %s", c.store.PaymentServiceUserID)
	return nil
}
