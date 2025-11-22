package instances

import (
	"fmt"
	"io"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/pkg"
)

type InstancesDescribeStore struct {
	WorkflowInstancesHistory []shared.V2WorkflowInstanceHistory `json:"workflowInstanceHistory"`
}
type InstancesDescribeController struct {
	store *InstancesDescribeStore
}

var _ fctl.Controller[*InstancesDescribeStore] = (*InstancesDescribeController)(nil)

func NewDefaultInstancesDescribeStore() *InstancesDescribeStore {
	return &InstancesDescribeStore{}
}

func NewInstancesDescribeController() *InstancesDescribeController {
	return &InstancesDescribeController{
		store: NewDefaultInstancesDescribeStore(),
	}
}

func NewDescribeCommand() *cobra.Command {
	c := NewInstancesDescribeController()
	return fctl.NewCommand("describe <instance-id>",
		fctl.WithShortDescription("Describe a specific workflow instance"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*InstancesDescribeStore](c),
	)
}

func (c *InstancesDescribeController) GetStore() *InstancesDescribeStore {
	return c.store
}

func (c *InstancesDescribeController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	response, err := store.Client().Orchestration.V2.GetInstanceHistory(cmd.Context(), operations.V2GetInstanceHistoryRequest{
		InstanceID: args[0],
	})
	if err != nil {
		return nil, err
	}

	c.store.WorkflowInstancesHistory = response.V2GetWorkflowInstanceHistoryResponse.Data

	return c, nil
}

func (c *InstancesDescribeController) Render(cmd *cobra.Command, args []string) error {
	store := fctl.GetStackStore(cmd.Context())
	for i, history := range c.store.WorkflowInstancesHistory {
		if err := printStage(cmd, i, store.Client(), args[0], history); err != nil {
			return err
		}
	}

	return nil
}

func printHistoryBaseInfo(out io.Writer, name string, ind int, history shared.V2WorkflowInstanceHistory) {
	fctl.Section.WithWriter(out).Printf("Stage %d : %s\n", ind, name)
	fctl.BasicText.WithWriter(out).Printfln("Started at: %s", history.StartedAt.Format(time.RFC3339))
	if history.Terminated {
		fctl.BasicText.WithWriter(out).Printfln("Terminated at: %s", history.TerminatedAt.Format(time.RFC3339))
	}
}

func stageSourceName(src *shared.V2StageSendSource) string {
	switch {
	case src.Wallet != nil:
		return fmt.Sprintf("wallet '%s' (balance: %s)", src.Wallet.ID, *src.Wallet.Balance)
	case src.Account != nil:
		return fmt.Sprintf("account '%s' (ledger: %s)", src.Account.ID, *src.Account.Ledger)
	case src.Payment != nil:
		return fmt.Sprintf("payment '%s'", src.Payment.ID)
	default:
		return "unknown_source_type"
	}
}

func stageDestinationName(dst *shared.V2StageSendDestination) string {
	switch {
	case dst.Wallet != nil:
		return fmt.Sprintf("wallet '%s' (balance: %s)", dst.Wallet.ID, *dst.Wallet.Balance)
	case dst.Account != nil:
		return fmt.Sprintf("account '%s' (ledger: %s)", dst.Account.ID, *dst.Account.Ledger)
	case dst.Payment != nil:
		return dst.Payment.Psp
	default:
		return "unknown_source_type"
	}
}

func subjectName(src shared.V2Subject) string {
	switch {
	case src.V2WalletSubject != nil:
		return fmt.Sprintf("wallet %s (balance: %s)", src.V2WalletSubject.Identifier, *src.V2WalletSubject.Balance)
	case src.V2LedgerAccountSubject != nil:
		return fmt.Sprintf("account %s", src.V2LedgerAccountSubject.Identifier)
	default:
		return "unknown_subject_type"
	}
}

func printMetadata(metadata map[string]string) []pterm.BulletListItem {
	ret := make([]pterm.BulletListItem, 0)
	ret = append(ret, historyItemDetails("Added metadata:"))
	for k, v := range metadata {
		ret = append(ret, pterm.BulletListItem{
			Level: 2,
			Text:  fmt.Sprintf("%s: %s", k, v),
		})
	}
	return ret
}

func printStage(cmd *cobra.Command, i int, client *formance.Formance, id string, history shared.V2WorkflowInstanceHistory) error {
	cyanWriter := fctl.BasicTextCyan
	defaultWriter := fctl.BasicText

	listItems := make([]pterm.BulletListItem, 0)

	switch history.Input.Type {
	case shared.V2StageTypeV2StageSend:
		printHistoryBaseInfo(cmd.OutOrStdout(), "send", i, history)
		if history.Input.V2StageSend != nil {
			cyanWriter.Printfln("Send %v %s from %s to %s", history.Input.V2StageSend.Amount.Amount,
				history.Input.V2StageSend.Amount.Asset, stageSourceName(history.Input.V2StageSend.Source),
				stageDestinationName(history.Input.V2StageSend.Destination))
		}
		fctl.Println()

		stageResponse, err := client.Orchestration.V2.GetInstanceStageHistory(cmd.Context(), operations.V2GetInstanceStageHistoryRequest{
			InstanceID: id,
			Number:     int64(i),
		})
		if err != nil {
			return err
		}

		for _, historyStage := range stageResponse.V2GetWorkflowInstanceHistoryStageResponse.Data {
			switch {
			case historyStage.Input.StripeTransfer != nil:
				listItems = append(listItems, historyItemTitle("Send %v %s to Stripe connected account: %s",
					*historyStage.Input.StripeTransfer.Amount,
					*historyStage.Input.StripeTransfer.Asset,
					*historyStage.Input.StripeTransfer.Destination,
				))
			case historyStage.Input.CreateTransaction != nil:
				if historyStage.Input.CreateTransaction.Data != nil && len(historyStage.Input.CreateTransaction.Data.Postings) > 0 {
					listItems = append(listItems, historyItemTitle("Send %v %s from account %s to account %s (ledger %s)",
						historyStage.Input.CreateTransaction.Data.Postings[0].Amount,
						historyStage.Input.CreateTransaction.Data.Postings[0].Asset,
						historyStage.Input.CreateTransaction.Data.Postings[0].Source,
						historyStage.Input.CreateTransaction.Data.Postings[0].Destination,
						*historyStage.Input.CreateTransaction.Ledger,
					))
				}
				if historyStage.Error == nil && historyStage.LastFailure == nil && historyStage.Terminated {
					if historyStage.Output.CreateTransaction != nil && len(historyStage.Output.CreateTransaction.Data) > 0 {
						txid := historyStage.Output.CreateTransaction.Data[0].Txid
						if txid != nil {
							listItems = append(listItems, historyItemDetails("Created transaction: %d", txid.Int64()))
						}
					}
					if historyStage.Input.CreateTransaction != nil && historyStage.Input.CreateTransaction.Data != nil && historyStage.Input.CreateTransaction.Data.Reference != nil {
						listItems = append(listItems, historyItemDetails("Reference: %s", *historyStage.Input.CreateTransaction.Data.Reference))
					}
					if historyStage.Input.CreateTransaction != nil && historyStage.Input.CreateTransaction.Data != nil && len(historyStage.Input.CreateTransaction.Data.Metadata) > 0 {
						listItems = append(listItems, printMetadata(historyStage.Input.CreateTransaction.Data.Metadata)...)
					}
				}
			case historyStage.Input.ConfirmHold != nil:
				listItems = append(listItems, historyItemTitle("Confirm debit hold %s", historyStage.Input.ConfirmHold.ID))
			case historyStage.Input.CreditWallet != nil:
				listItems = append(listItems, historyItemTitle("Credit wallet %s (balance: %s) of %v %s from %s",
					*historyStage.Input.CreditWallet.ID,
					*historyStage.Input.CreditWallet.Data.Balance,
					historyStage.Input.CreditWallet.Data.Amount.Amount,
					historyStage.Input.CreditWallet.Data.Amount.Asset,
					subjectName(historyStage.Input.CreditWallet.Data.Sources[0]),
				))
				if historyStage.Error == nil && historyStage.LastFailure == nil && historyStage.Terminated {
					if len(historyStage.Input.CreditWallet.Data.Metadata) > 0 {
						listItems = append(listItems, printMetadata(historyStage.Input.CreditWallet.Data.Metadata)...)
					}
				}
			case historyStage.Input.DebitWallet != nil:
				destination := "@world"
				if historyStage.Input.DebitWallet.Data.Destination != nil {
					destination = subjectName(*historyStage.Input.DebitWallet.Data.Destination)
				}

				listItems = append(listItems, historyItemTitle("Debit wallet %s (balance: %s) of %v %s to %s",
					*historyStage.Input.DebitWallet.ID,
					historyStage.Input.DebitWallet.Data.Balances[0],
					historyStage.Input.DebitWallet.Data.Amount.Amount,
					historyStage.Input.DebitWallet.Data.Amount.Asset,
					destination,
				))
				if historyStage.Error == nil && historyStage.LastFailure == nil && historyStage.Terminated {
					if len(historyStage.Input.DebitWallet.Data.Metadata) > 0 {
						listItems = append(listItems, printMetadata(historyStage.Input.DebitWallet.Data.Metadata)...)
					}
				}
			case historyStage.Input.GetAccount != nil:
				listItems = append(listItems, historyItemTitle("Read account %s of ledger %s",
					historyStage.Input.GetAccount.ID,
					historyStage.Input.GetAccount.Ledger,
				))
			case historyStage.Input.GetPayment != nil:
				listItems = append(listItems, historyItemTitle("Read payment %s",
					historyStage.Input.GetPayment.ID))
			case historyStage.Input.GetWallet != nil:
				listItems = append(listItems, historyItemTitle("Read wallet '%s'", historyStage.Input.GetWallet.ID))
			// V2 doesn't have RevertTransaction, skip it
			case historyStage.Input.VoidHold != nil:
				listItems = append(listItems, historyItemTitle("Cancel debit hold %s", historyStage.Input.VoidHold.ID))
			case historyStage.Input.ListWallets != nil:
				listItems = append(listItems, historyItemTitle("List wallets"))
			}
			if historyStage.LastFailure != nil {
				listItems = append(listItems, historyItemError(*historyStage.LastFailure))
				if historyStage.NextExecution != nil {
					listItems = append(listItems, historyItemError("Next try: %s", historyStage.NextExecution.Format(time.RFC3339)))
					listItems = append(listItems, historyItemError("Attempt: %d", historyStage.Attempt))
				}
			}
			if historyStage.Error != nil {
				listItems = append(listItems, historyItemError(*historyStage.Error))
			}
		}
	case shared.V2StageTypeV2StageDelay:
		printHistoryBaseInfo(cmd.OutOrStdout(), "delay", i, history)
		if history.Input.V2StageDelay != nil {
			switch {
			case history.Input.V2StageDelay.Duration != nil:
				listItems = append(listItems, historyItemTitle("Pause workflow for a delay of %s", *history.Input.V2StageDelay.Duration))
			case history.Input.V2StageDelay.Until != nil:
				listItems = append(listItems, historyItemTitle("Pause workflow until %s", *history.Input.V2StageDelay.Until))
			}
		}
	case shared.V2StageTypeV2StageWaitEvent:
		printHistoryBaseInfo(cmd.OutOrStdout(), "wait_event", i, history)
		if history.Input.V2StageWaitEvent != nil {
			listItems = append(listItems, historyItemTitle("Waiting event '%s'", history.Input.V2StageWaitEvent.Event))
		}
		if history.Error == nil {
			if history.Terminated {
				listItems = append(listItems, historyItemDetails("Event received!"))
			} else {
				listItems = append(listItems, historyItemDetails("Still waiting event..."))
			}
		}
	case shared.V2StageTypeV2Update:
		printHistoryBaseInfo(cmd.OutOrStdout(), "update", i, history)
		if history.Input.V2Update != nil {
			switch {
			case history.Input.V2Update.Account != nil:
				account := history.Input.V2Update.Account
				listItems = append(listItems, historyItemTitle("Update account '%s' of ledger '%s'", account.ID, account.Ledger))
				listItems = append(listItems, printMetadata(account.Metadata)...)
			}
		}
	default:
		// Display error?
	}
	if history.Error != nil {
		fctl.BasicTextRed.WithWriter(cmd.OutOrStdout()).Printfln("Stage terminated with error: %s", *history.Error)
	}

	if len(listItems) > 0 {
		defaultWriter.Print("History :\n")
		return pterm.DefaultBulletList.WithWriter(cmd.OutOrStdout()).WithItems(listItems).Render()
	}
	return nil
}

func historyItemTitle(format string, args ...any) pterm.BulletListItem {
	return pterm.BulletListItem{
		Level:     0,
		TextStyle: fctl.StyleGreen,
		Text:      fmt.Sprintf(format, args...),
	}
}

func historyItemDetails(format string, args ...any) pterm.BulletListItem {
	return pterm.BulletListItem{
		Level: 1,
		Text:  fmt.Sprintf(format, args...),
	}
}

func historyItemError(format string, args ...any) pterm.BulletListItem {
	return pterm.BulletListItem{
		Level:     1,
		TextStyle: fctl.StyleRed,
		Text:      fmt.Sprintf(format, args...),
	}
}
