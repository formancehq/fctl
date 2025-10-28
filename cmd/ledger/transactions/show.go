package transactions

import (
	"github.com/formancehq/fctl/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/spf13/cobra"
)

type ShowStore struct {
	Transaction shared.Transaction `json:"transaction"`
}
type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewDefaultShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("show <transaction-id>",
		fctl.WithShortDescription("Print a transaction"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithAliases("sh"),
		fctl.WithValidArgs("last"),
		fctl.WithController[*ShowStore](NewShowController()),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	ledger := fctl.GetString(cmd, internal.LedgerFlag)
	txId, err := internal.TransactionIDOrLastN(cmd.Context(), stackClient, ledger, args[0])
	if err != nil {
		return nil, err
	}

	response, err := stackClient.Ledger.V1.GetTransaction(cmd.Context(), operations.GetTransactionRequest{
		Ledger: ledger,
		Txid:   txId,
	})
	if err != nil {
		return nil, err
	}

	c.store.Transaction = response.TransactionResponse.Data

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintExpandedTransaction(cmd.OutOrStdout(), internal.WrapV1Transaction(c.store.Transaction))
}
