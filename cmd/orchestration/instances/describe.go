package instances

import (
	"fmt"
	"io"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	formance "github.com/formancehq/formance-sdk-go/v4"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/orchestration"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type InstancesDescribeStore struct {
	WorkflowInstancesHistory []orchestration.WorkflowInstanceHistory `json:"workflowInstanceHistory"`
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

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	response, err := stackClient.Orchestration.V1.GetInstanceHistory(cmd.Context(), operations.GetInstanceHistoryRequest{
		InstanceID: args[0],
	})
	if err != nil {
		return nil, err
	}

	c.store.WorkflowInstancesHistory = response.GetWorkflowInstanceHistoryResponse.WorkflowInstanceHistoryList

	return c, nil
}

func (c *InstancesDescribeController) Render(cmd *cobra.Command, args []string) error {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return err
	}

	for i, history := range c.store.WorkflowInstancesHistory {
		if err := printStage(cmd, i, stackClient, args[0], history); err != nil {
			return err
		}
	}

	return nil
}

func printHistoryBaseInfo(out io.Writer, name string, ind int, history orchestration.WorkflowInstanceHistory) {
	fctl.Section.WithWriter(out).Printf("Stage %d : %s\n", ind, name)
	fctl.BasicText.WithWriter(out).Printfln("Started at: %s", history.StartedAt.Format(time.RFC3339))
	if history.Terminated {
		fctl.BasicText.WithWriter(out).Printfln("Terminated at: %s", history.StartedAt.Format(time.RFC3339))
	}
}

func stageSourceName(src *orchestration.StageSendSource) string {
	switch {
	case src.StageSendSourceWallet != nil:
		return fmt.Sprintf("wallet '%s' (balance: %s)", src.StageSendSourceWallet.ID, *src.StageSendSourceWallet.Balance)
	case src.StageSendSourceAccount != nil:
		return fmt.Sprintf("account '%s' (ledger: %s)", src.StageSendSourceAccount.ID, *src.StageSendSourceAccount.Ledger)
	case src.StageSendSourcePayment != nil:
		return fmt.Sprintf("payment '%s'", src.StageSendSourcePayment.ID)
	default:
		return "unknown_source_type"
	}
}

func stageDestinationName(dst *orchestration.StageSendDestination) string {
	switch {
	case dst.StageSendSourceWallet != nil:
		return fmt.Sprintf("wallet '%s' (balance: %s)", dst.StageSendSourceWallet.ID, *dst.StageSendSourceWallet.Balance)
	case dst.StageSendSourceAccount != nil:
		return fmt.Sprintf("account '%s' (ledger: %s)", dst.StageSendSourceAccount.ID, *dst.StageSendSourceAccount.Ledger)
	case dst.StageSendDestinationPayment != nil:
		return dst.StageSendDestinationPayment.Psp
	default:
		return "unknown_source_type"
	}
}

func subjectName(src orchestration.Subject) string {
	switch {
	case src.WalletSubject != nil:
		return fmt.Sprintf("wallet %s (balance: %s)", src.WalletSubject.Identifier, *src.WalletSubject.Balance)
	case src.LedgerAccountSubject != nil:
		return fmt.Sprintf("account %s", src.LedgerAccountSubject.Identifier)
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

func printStage(cmd *cobra.Command, i int, client *formance.Formance, id string, history orchestration.WorkflowInstanceHistory) error {
	cyanWriter := fctl.BasicTextCyan
	defaultWriter := fctl.BasicText

	listItems := make([]pterm.BulletListItem, 0)

	switch history.Stage.Type {
	case orchestration.StageTypeStageSend:
		printHistoryBaseInfo(cmd.OutOrStdout(), "send", i, history)
		cyanWriter.Printfln("Send %v %s from %s to %s", history.Stage.StageSend.Monetary.Amount,
			history.Stage.StageSend.Monetary.Asset, stageSourceName(history.Stage.StageSend.StageSendSource),
			stageDestinationName(history.Stage.StageSend.StageSendDestination))
		fctl.Println()

		stageResponse, err := client.Orchestration.V1.GetInstanceStageHistory(cmd.Context(), operations.GetInstanceStageHistoryRequest{
			InstanceID: id,
			Number:     int64(i),
		})
		if err != nil {
			return err
		}

		for _, historyStage := range stageResponse.GetWorkflowInstanceHistoryStageResponse.WorkflowInstanceHistoryStageList {
			switch {
			case historyStage.WorkflowInstanceHistoryStageInput.StripeTransferRequest != nil:
				listItems = append(listItems, historyItemTitle("Send %v %s to Stripe connected account: %s",
					*historyStage.WorkflowInstanceHistoryStageInput.StripeTransferRequest.Amount,
					*historyStage.WorkflowInstanceHistoryStageInput.StripeTransferRequest.Asset,
					*historyStage.WorkflowInstanceHistoryStageInput.StripeTransferRequest.Destination,
				))
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction != nil:
				listItems = append(listItems, historyItemTitle("Send %v %s from account %s to account %s (ledger %s)",
					historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.PostTransaction.Postings[0].Amount,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.PostTransaction.Postings[0].Asset,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.PostTransaction.Postings[0].Source,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.PostTransaction.Postings[0].Destination,
					*historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.Ledger,
				))
				if historyStage.Error == nil && historyStage.LastFailure == nil && historyStage.Terminated {
					listItems = append(listItems, historyItemDetails("Created transaction: %d", historyStage.WorkflowInstanceHistoryStageOutput.CreateTransactionResponse.Transaction.ID))
					if historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.PostTransaction.Reference != nil {
						listItems = append(listItems, historyItemDetails("Reference: %s", *historyStage.WorkflowInstanceHistoryStageOutput.CreateTransactionResponse.Transaction.Reference))
					}
					if len(historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.PostTransaction.Metadata) > 0 {
						listItems = append(listItems, printMetadata(historyStage.WorkflowInstanceHistoryStageInput.ActivityCreateTransaction.PostTransaction.Metadata)...)
					}
				}
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityConfirmHold != nil:
				listItems = append(listItems, historyItemTitle("Confirm debit hold %s", historyStage.WorkflowInstanceHistoryStageInput.ActivityConfirmHold.ID))
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet != nil:
				listItems = append(listItems, historyItemTitle("Credit wallet %s (balance: %s) of %v %s from %s",
					*historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet.ID,
					*historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet.CreditWalletRequest.Balance,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet.CreditWalletRequest.Monetary.Amount,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet.CreditWalletRequest.Monetary.Asset,
					subjectName(historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet.CreditWalletRequest.Sources[0]),
				))
				if historyStage.Error == nil && historyStage.LastFailure == nil && historyStage.Terminated {
					if len(historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet.CreditWalletRequest.Metadata) > 0 {
						listItems = append(listItems, printMetadata(historyStage.WorkflowInstanceHistoryStageInput.ActivityCreditWallet.CreditWalletRequest.Metadata)...)
					}
				}
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet != nil:
				destination := "@world"
				if historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.DebitWalletRequest.Subject != nil {
					destination = subjectName(*historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.DebitWalletRequest.Subject)
				}

				listItems = append(listItems, historyItemTitle("Debit wallet %s (balance: %s) of %v %s to %s",
					*historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.ID,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.DebitWalletRequest.Balances[0],
					historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.DebitWalletRequest.Monetary.Amount,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.DebitWalletRequest.Monetary.Asset,
					destination,
				))
				if historyStage.Error == nil && historyStage.LastFailure == nil && historyStage.Terminated {
					if len(historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.DebitWalletRequest.Metadata) > 0 {
						listItems = append(listItems, printMetadata(historyStage.WorkflowInstanceHistoryStageInput.ActivityDebitWallet.DebitWalletRequest.Metadata)...)
					}
				}
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityGetAccount != nil:
				listItems = append(listItems, historyItemTitle("Read account %s of ledger %s",
					historyStage.WorkflowInstanceHistoryStageInput.ActivityGetAccount.ID,
					historyStage.WorkflowInstanceHistoryStageInput.ActivityGetAccount.Ledger,
				))
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityGetPayment != nil:
				listItems = append(listItems, historyItemTitle("Read payment %s",
					historyStage.WorkflowInstanceHistoryStageInput.ActivityGetPayment.ID))
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityGetWallet != nil:
				listItems = append(listItems, historyItemTitle("Read wallet '%s'", historyStage.WorkflowInstanceHistoryStageInput.ActivityGetWallet.ID))
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityRevertTransaction != nil:
				listItems = append(listItems, historyItemTitle("Revert transaction %s", historyStage.WorkflowInstanceHistoryStageInput.ActivityRevertTransaction.ID))
				if historyStage.Error == nil {
					listItems = append(listItems, historyItemTitle("Created transaction: %d", historyStage.WorkflowInstanceHistoryStageOutput.CreateTransactionResponse1.Transaction.ID))
				}
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityVoidHold != nil:
				listItems = append(listItems, historyItemTitle("Cancel debit hold %s", historyStage.WorkflowInstanceHistoryStageInput.ActivityVoidHold.ID))
			case historyStage.WorkflowInstanceHistoryStageInput.ActivityListWallets != nil:
				listItems = append(listItems, historyItemTitle("List wallets"))
			}
			if historyStage.LastFailure != nil {
				listItems = append(listItems, historyItemError("%s", *historyStage.LastFailure))
				if historyStage.NextExecution != nil {
					listItems = append(listItems, historyItemError("Next try: %s", historyStage.NextExecution.Format(time.RFC3339)))
					listItems = append(listItems, historyItemError("Attempt: %d", historyStage.Attempt))
				}
			}
			if historyStage.Error != nil {
				listItems = append(listItems, historyItemError("%s", *historyStage.Error))
			}
		}
	case orchestration.StageTypeStageDelay:
		printHistoryBaseInfo(cmd.OutOrStdout(), "delay", i, history)
		switch {
		case history.Stage.StageDelay.Duration != nil:
			listItems = append(listItems, historyItemTitle("Pause workflow for a delay of %s", *history.Stage.StageDelay.Duration))
		case history.Stage.StageDelay.Until != nil:
			listItems = append(listItems, historyItemTitle("Pause workflow until %s", *history.Stage.StageDelay.Until))
		}
	case orchestration.StageTypeStageWaitEvent:
		printHistoryBaseInfo(cmd.OutOrStdout(), "wait_event", i, history)
		listItems = append(listItems, historyItemTitle("Waiting event '%s'", history.Stage.StageWaitEvent.Event))
		if history.Error == nil {
			if history.Terminated {
				listItems = append(listItems, historyItemDetails("Event received!"))
			} else {
				listItems = append(listItems, historyItemDetails("Still waiting event..."))
			}
		}
	case orchestration.StageTypeUpdate:
		printHistoryBaseInfo(cmd.OutOrStdout(), "update", i, history)
		switch {
		case history.Stage.Update.UpdateAccount != nil:
			account := history.Stage.Update.UpdateAccount
			listItems = append(listItems, historyItemTitle("Update account '%s' of ledger '%s'", account.ID, account.Ledger))
			listItems = append(listItems, printMetadata(account.Metadata)...)
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
