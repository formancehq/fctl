package transactions

import (
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/fctl/v3/cmd/ledger/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type RevertStore struct {
	Transaction internal.Transaction `json:"transaction"`
}
type RevertController struct {
	store *RevertStore
}

var _ fctl.Controller[*RevertStore] = (*RevertController)(nil)

func NewDefaultRevertStore() *RevertStore {
	return &RevertStore{}
}

func NewRevertController() *RevertController {
	return &RevertController{
		store: NewDefaultRevertStore(),
	}
}

func NewRevertCommand() *cobra.Command {
	return fctl.NewCommand("revert <transaction-id>",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Revert a transaction"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgs("last"),
		fctl.WithBoolFlag("at-effective-date", false, "set the timestamp to the original transaction timestamp"),
		fctl.WithBoolFlag("force", false, "Force the revert even if the account does not have enough funds"),
		fctl.WithController[*RevertStore](NewRevertController()),
	)
}

func (c *RevertController) GetStore() *RevertStore {
	return c.store
}

func (c *RevertController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to revert transaction %s", args[0]) {
		return nil, fctl.ErrMissingApproval
	}

	ledger := fctl.GetString(cmd, internal.LedgerFlag)
	txId, err := internal.TransactionIDOrLastN(cmd.Context(), stackClient, ledger, args[0])
	if err != nil {
		return nil, err
	}

	force := fctl.GetBool(cmd, "force")

	if fctl.GetBool(cmd, "at-effective-date") {
		request := operations.V2RevertTransactionRequest{
			Ledger:          ledger,
			ID:              txId,
			AtEffectiveDate: pointer.For(true),
			Force:           &force,
		}

		response, err := stackClient.Ledger.V2.RevertTransaction(cmd.Context(), request)
		if err != nil {
			return nil, err
		}

		c.store.Transaction = internal.WrapV2Transaction(response.V2CreateTransactionResponse.Data)
	} else {
		request := operations.RevertTransactionRequest{
			Ledger:        ledger,
			Txid:          txId,
			DisableChecks: &force,
		}

		response, err := stackClient.Ledger.V1.RevertTransaction(cmd.Context(), request)
		if err != nil {
			return nil, err
		}

		c.store.Transaction = internal.WrapV1Transaction(response.TransactionResponse.Data)
	}

	return c, nil
}

func (c *RevertController) Render(cmd *cobra.Command, args []string) error {
	return internal.PrintTransaction(cmd.OutOrStdout(), c.store.Transaction)
}
