package bankaccounts

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
)

type UpdateMetadataStore struct {
	Success bool `json:"success"`
}
type UpdateMetadataController struct {
	PaymentsVersion versions.Version

	store *UpdateMetadataStore
}

func (c *UpdateMetadataController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*UpdateMetadataStore] = (*UpdateMetadataController)(nil)

func NewUpdateMetadataStore() *UpdateMetadataStore {
	return &UpdateMetadataStore{}
}

func NewUpdateMetadataController() *UpdateMetadataController {
	return &UpdateMetadataController{
		store: NewUpdateMetadataStore(),
	}
}

func NewUpdateMetadataCommand() *cobra.Command {
	c := NewUpdateMetadataController()
	return fctl.NewCommand("update-metadata <bankAccountID> [<key>=<value>...]",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Set metadata on bank account"),
		fctl.WithAliases("um", "update-meta"),
		fctl.WithArgs(cobra.MinimumNArgs(2)),
		fctl.WithController[*UpdateMetadataStore](c),
	)
}

func (c *UpdateMetadataController) GetStore() *UpdateMetadataStore {
	return c.store
}

func (c *UpdateMetadataController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

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

	if c.PaymentsVersion < versions.V1 {
		return nil, fmt.Errorf("bank accounts are only supported in >= v1.0.0")
	}

	metadata, err := fctl.ParseMetadata(args[1:])
	if err != nil {
		return nil, err
	}

	bankAccountID := args[0]

	if !fctl.CheckStackApprobation(cmd, "You are about to set a metadata on bank account '%s'", bankAccountID) {
		return nil, fctl.ErrMissingApproval
	}
	if c.PaymentsVersion >= versions.V3 {
		request := operations.V3UpdateBankAccountMetadataRequest{
			V3UpdateBankAccountMetadataRequest: &shared.V3UpdateBankAccountMetadataRequest{
				Metadata: metadata,
			},
			BankAccountID: bankAccountID,
		}

		response, err := stackClient.Payments.V3.UpdateBankAccountMetadata(cmd.Context(), request)
		if err != nil {
			return nil, err
		}

		if response.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}

		c.store.Success = response.StatusCode == 204
		return c, nil
	}

	request := operations.UpdateBankAccountMetadataRequest{
		UpdateBankAccountMetadataRequest: shared.UpdateBankAccountMetadataRequest{
			Metadata: metadata,
		},
		BankAccountID: bankAccountID,
	}

	response, err := stackClient.Payments.V1.UpdateBankAccountMetadata(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Success = response.StatusCode == 204

	return c, nil
}

func (c *UpdateMetadataController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Metadata added!")
	return nil
}
