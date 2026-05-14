package cmd

import (
	"context"
	"fmt"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	ledgercmd "github.com/formancehq/fctl/v4/internal/commands/ledger"
	"github.com/formancehq/fctl/v4/internal/render"
)

func newLedgerCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "ledger",
		Short: "Manage ledgers",
	}
	command.AddCommand(newLedgerTransactionsCommand())
	return command
}

func newLedgerTransactionsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "transactions",
		Short: "Manage ledger transactions",
	}
	command.AddCommand(newLedgerTransactionsListCommand())
	return command
}

func newLedgerTransactionsListCommand() *cobra.Command {
	var ledger string
	var pageSize int64
	var cursor string
	var account string
	var source string
	var destination string
	var reference string
	var apiVersion string

	command := &cobra.Command{
		Use:   "list",
		Short: "List ledger transactions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}
			if ledger == "" {
				ledger = rt.Context.Defaults["ledger"]
			}
			if ledger == "" {
				ledger = "default"
			}

			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(
				formance.WithServerURL(rt.Target.URL),
				formance.WithClient(httpClient),
			)
			service := ledgercmd.ListTransactionsService{
				Handlers: ledgercmd.SDKListTransactionsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureListTransactions,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.ListTransactionsInput{
				Ledger:      ledger,
				PageSize:    pageSize,
				Cursor:      cursor,
				Account:     account,
				Source:      source,
				Destination: destination,
				Reference:   reference,
			})
			if err != nil {
				return err
			}

			format, err := outputFormat(cmd)
			if err != nil {
				return err
			}
			if format == "json" {
				return render.JSON(cmd.OutOrStdout(), output)
			}
			return renderLedgerTransactions(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&account, "account", "", "Filter by account")
	command.Flags().StringVar(&source, "src", "", "Filter by source account")
	command.Flags().StringVar(&destination, "dst", "", "Filter by destination account")
	command.Flags().StringVar(&reference, "reference", "", "Filter by reference")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func renderLedgerTransactions(cmd *cobra.Command, output ledgercmd.ListTransactionsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Transactions) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No transactions found.")
		return err
	}
	for _, transaction := range output.Transactions {
		reference := ""
		if transaction.Reference != nil {
			reference = *transaction.Reference
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n",
			transaction.ID,
			reference,
			transaction.Timestamp.Format(time.RFC3339),
		); err != nil {
			return err
		}
	}
	return nil
}
