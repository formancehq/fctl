package psu

import (
	"errors"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateLinkStore struct {
	AttemptID string `json:"attemptID"`
	Link      string `json:"link"`
}

type CreateLinkController struct {
	PaymentsVersion versions.Version
	store           *CreateLinkStore

	clientRedirectURLFlag string
	applicationNameFlag   string
}

func (c *CreateLinkController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*CreateLinkStore] = (*CreateLinkController)(nil)

func NewCreateLinkController() *CreateLinkController {
	return &CreateLinkController{
		store:                 &CreateLinkStore{},
		clientRedirectURLFlag: "client-redirect-url",
		applicationNameFlag:   "application-name",
	}
}

func (c *CreateLinkController) GetStore() *CreateLinkStore {
	return c.store
}

func NewCreateLinkCommand() *cobra.Command {
	c := NewCreateLinkController()
	return fctl.NewCommand("create-link <paymentServiceUserID> <connectorID>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Create an open banking user link for a payment service user"),
		fctl.WithAliases("link"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(c.clientRedirectURLFlag, "", "URL to redirect the user to once the link flow completes (required)"),
		fctl.WithStringFlag(c.applicationNameFlag, "", "Application name shown to the user during the link flow (some providers require this)"),
		fctl.WithController[*CreateLinkStore](c),
	)
}

func (c *CreateLinkController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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

	if c.PaymentsVersion.Major < versions.V3 {
		return nil, fmt.Errorf("payment service users require Payments API v3")
	}

	paymentServiceUserID := args[0]
	if paymentServiceUserID == "" {
		return nil, errors.New("payment service user ID is required")
	}

	connectorID := args[1]
	if connectorID == "" {
		return nil, errors.New("connector ID is required")
	}

	clientRedirectURL := fctl.GetString(cmd, c.clientRedirectURLFlag)
	if clientRedirectURL == "" {
		return nil, errors.New("--client-redirect-url is required")
	}

	var applicationName *string
	if name := fctl.GetString(cmd, c.applicationNameFlag); name != "" {
		applicationName = &name
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to create an open banking link for payment service user '%s' on connector '%s'", paymentServiceUserID, connectorID) {
		return nil, fctl.ErrMissingApproval
	}

	response, err := stackClient.Payments.V3.CreateLinkForPaymentServiceUser(cmd.Context(), operations.V3CreateLinkForPaymentServiceUserRequest{
		PaymentServiceUserID: paymentServiceUserID,
		ConnectorID:          connectorID,
		V3PaymentServiceUserCreateLinkRequest: &payments.V3PaymentServiceUserCreateLinkRequest{
			ClientRedirectURL: clientRedirectURL,
			ApplicationName:   applicationName,
		},
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}
	if response.V3PaymentServiceUserCreateLinkResponse == nil {
		return nil, fmt.Errorf("unexpected empty response")
	}

	c.store.AttemptID = response.V3PaymentServiceUserCreateLinkResponse.AttemptID
	c.store.Link = response.V3PaymentServiceUserCreateLinkResponse.Link

	return c, nil
}

func (c *CreateLinkController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Open banking link created.")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("AttemptID"), c.store.AttemptID})
	tableData = append(tableData, []string{pterm.LightCyan("Link"), c.store.Link})

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
