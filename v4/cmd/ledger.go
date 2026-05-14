package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	ledgercmd "github.com/formancehq/fctl/v4/internal/commands/ledger"
)

func newLedgerCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "ledger",
		Short: "Manage ledgers",
	}
	command.AddCommand(newLedgerInfoCommand("info", nil, false))
	command.AddCommand(newLedgerInfoCommand("server-infos", []string{"si"}, true))
	command.AddCommand(newLedgerAccountsCommand())
	command.AddCommand(newLedgerListCommand())
	command.AddCommand(newLedgerStatsCommand())
	command.AddCommand(newLedgerTransactionsCommand())
	command.AddCommand(newLedgerVolumesCommand())
	return command
}

func newLedgerAccountsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "accounts",
		Short: "Manage ledger accounts",
	}
	command.AddCommand(newLedgerAccountsListCommand())
	command.AddCommand(newLedgerAccountsShowCommand())
	return command
}

func newLedgerAccountsListCommand() *cobra.Command {
	var ledger string
	var pageSize int64
	var cursor string
	var account string
	var metadata []string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List ledger accounts",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("address") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --address has been deprecated, use --account")
			}

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

			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}

			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(
				formance.WithServerURL(rt.Target.URL),
				formance.WithClient(httpClient),
			)
			service := ledgercmd.ListAccountsService{
				Handlers: ledgercmd.SDKListAccountsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureListAccounts,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.ListAccountsInput{
				Ledger:   ledger,
				PageSize: pageSize,
				Cursor:   cursor,
				Account:  account,
				Metadata: parsedMetadata,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerAccounts(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&account, "account", "", "Filter by account address")
	command.Flags().StringVar(&account, "address", "", "Deprecated alias for --account")
	command.Flags().StringSliceVar(&metadata, "metadata", nil, "Filter by metadata key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")
	_ = command.Flags().MarkDeprecated("address", "use --account")

	return command
}

func newLedgerAccountsShowCommand() *cobra.Command {
	var ledger string
	var apiVersion string

	command := &cobra.Command{
		Use:     "show <account>",
		Aliases: []string{"sh", "s"},
		Short:   "Show a ledger account",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			service := ledgercmd.GetAccountService{
				Handlers: ledgercmd.SDKGetAccountHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureGetAccount,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.GetAccountInput{
				Ledger:  ledger,
				Account: args[0],
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerAccount(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerInfoCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Read ledger server info",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deprecated {
				fmt.Fprintf(cmd.ErrOrStderr(), "Command ledger %s has been deprecated, use ledger info\n", cmd.CalledAs())
			}

			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}

			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(
				formance.WithServerURL(rt.Target.URL),
				formance.WithClient(httpClient),
			)
			service := ledgercmd.ReadInfoService{
				Handlers: ledgercmd.SDKReadInfoHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureGetInfo,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context())
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerInfo(cmd, output)
		},
	}

	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")
	if deprecated {
		command.Deprecated = "use ledger info"
	}

	return command
}

func newLedgerListCommand() *cobra.Command {
	var pageSize int64
	var cursor string
	var includeDeleted bool
	var sort string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List ledgers",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := runtimeFromCommand(cmd)
			if err != nil {
				return err
			}

			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(
				formance.WithServerURL(rt.Target.URL),
				formance.WithClient(httpClient),
			)
			service := ledgercmd.ListLedgersService{
				Handlers: ledgercmd.SDKListLedgersHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureListLedgers,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.ListLedgersInput{
				PageSize:       pageSize,
				Cursor:         cursor,
				IncludeDeleted: includeDeleted,
				Sort:           sort,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgers(cmd, output)
		},
	}

	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().BoolVar(&includeDeleted, "include-deleted", false, "Include deleted ledgers")
	command.Flags().StringVar(&sort, "sort", "", "Sort expression, formatted as <field>:<asc|desc>")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerStatsCommand() *cobra.Command {
	var ledger string
	var apiVersion string

	command := &cobra.Command{
		Use:     "stats",
		Aliases: []string{"st"},
		Short:   "Read ledger stats",
		Args:    cobra.NoArgs,
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
			service := ledgercmd.ReadStatsService{
				Handlers: ledgercmd.SDKReadStatsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureReadStats,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.ReadStatsInput{Ledger: ledger})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerStats(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerTransactionsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "transactions",
		Short: "Manage ledger transactions",
	}
	command.AddCommand(newLedgerTransactionsListCommand())
	command.AddCommand(newLedgerTransactionsShowCommand())
	return command
}

func newLedgerVolumesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "volumes",
		Short: "Manage ledger volumes",
	}
	command.AddCommand(newLedgerVolumesListCommand())
	return command
}

func newLedgerVolumesListCommand() *cobra.Command {
	var ledger string
	var pageSize int64
	var cursor string
	var account string
	var metadata []string
	var startTime string
	var endTime string
	var useInsertionDate bool
	var groupBy int64
	var sort string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List ledger volumes and balances",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("address") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --address has been deprecated, use --account")
			}
			if cmd.Flags().Changed("oot") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --oot has been deprecated, use --start-time")
			}
			if cmd.Flags().Changed("pit") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --pit has been deprecated, use --end-time")
			}
			if cmd.Flags().Changed("insertion-date") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --insertion-date has been deprecated, use --use-insertion-date")
			}

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

			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			parsedStartTime, err := parseOptionalRFC3339(startTime, "start-time")
			if err != nil {
				return err
			}
			parsedEndTime, err := parseOptionalRFC3339(endTime, "end-time")
			if err != nil {
				return err
			}

			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(
				formance.WithServerURL(rt.Target.URL),
				formance.WithClient(httpClient),
			)
			service := ledgercmd.ListVolumesService{
				Handlers: ledgercmd.SDKListVolumesHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureGetVolumesWithBalances,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.ListVolumesInput{
				Ledger:           ledger,
				PageSize:         pageSize,
				Cursor:           cursor,
				Account:          account,
				Metadata:         parsedMetadata,
				StartTime:        parsedStartTime,
				EndTime:          parsedEndTime,
				UseInsertionDate: useInsertionDate,
				GroupBy:          groupBy,
				Sort:             sort,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerVolumes(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().Int64Var(&pageSize, "page-size", 10, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&account, "account", "", "Filter by account address")
	command.Flags().StringVar(&account, "address", "", "Deprecated alias for --account")
	command.Flags().StringSliceVar(&metadata, "metadata", nil, "Filter by metadata key=value")
	command.Flags().StringVar(&startTime, "start-time", "", "Origin of time in RFC3339 format")
	command.Flags().StringVar(&startTime, "oot", "", "Deprecated alias for --start-time")
	command.Flags().StringVar(&endTime, "end-time", "", "Point in time in RFC3339 format")
	command.Flags().StringVar(&endTime, "pit", "", "Deprecated alias for --end-time")
	command.Flags().BoolVar(&useInsertionDate, "use-insertion-date", false, "Use insertion date")
	command.Flags().BoolVar(&useInsertionDate, "insertion-date", false, "Deprecated alias for --use-insertion-date")
	command.Flags().Int64Var(&groupBy, "group-by", 0, "Group by address segment level")
	command.Flags().StringVar(&sort, "sort", "", "Sort expression, formatted as <field>:<asc|desc>")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")
	_ = command.Flags().MarkDeprecated("address", "use --account")
	_ = command.Flags().MarkDeprecated("oot", "use --start-time")
	_ = command.Flags().MarkDeprecated("pit", "use --end-time")
	_ = command.Flags().MarkDeprecated("insertion-date", "use --use-insertion-date")

	return command
}

func newLedgerTransactionsShowCommand() *cobra.Command {
	var ledger string
	var apiVersion string

	command := &cobra.Command{
		Use:     "show <transaction-id>",
		Aliases: []string{"sh"},
		Short:   "Show a ledger transaction",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			service := ledgercmd.GetTransactionService{
				Handlers: ledgercmd.SDKGetTransactionHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureGetTransaction,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.GetTransactionInput{
				Ledger:        ledger,
				TransactionID: args[0],
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerTransaction(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

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
			if cmd.Flags().Changed("src") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --src has been deprecated, use --source")
			}
			if cmd.Flags().Changed("dst") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --dst has been deprecated, use --destination")
			}

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

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerTransactions(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&account, "account", "", "Filter by account")
	command.Flags().StringVar(&source, "source", "", "Filter by source account")
	command.Flags().StringVar(&source, "src", "", "Deprecated alias for --source")
	command.Flags().StringVar(&destination, "destination", "", "Filter by destination account")
	command.Flags().StringVar(&destination, "dst", "", "Deprecated alias for --destination")
	command.Flags().StringVar(&reference, "reference", "", "Filter by reference")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")
	_ = command.Flags().MarkDeprecated("src", "use --source")
	_ = command.Flags().MarkDeprecated("dst", "use --destination")

	return command
}

func renderLedgers(cmd *cobra.Command, output ledgercmd.ListLedgersOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Ledgers) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No ledgers found.")
		return err
	}
	for _, ledger := range output.Ledgers {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n",
			ledger.Name,
			ledger.Bucket,
			ledger.AddedAt.Format(time.RFC3339),
		); err != nil {
			return err
		}
	}
	return nil
}

func renderLedgerAccounts(cmd *cobra.Command, output ledgercmd.ListAccountsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Accounts) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No accounts found.")
		return err
	}
	for _, account := range output.Accounts {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), account.Address); err != nil {
			return err
		}
	}
	return nil
}

func renderLedgerAccount(cmd *cobra.Command, output ledgercmd.GetAccountOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Address\t%s\n", output.Account.Address); err != nil {
		return err
	}
	if len(output.Account.Volumes) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No volumes.")
		return err
	}
	for asset, volume := range output.Account.Volumes {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\tinput=%s\toutput=%s\tbalance=%s\n",
			asset, volume.Input, volume.Output, volume.Balance); err != nil {
			return err
		}
	}
	return nil
}

func renderLedgerInfo(cmd *cobra.Command, output ledgercmd.ReadInfoOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Server\t%s\n", output.Server); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Version\t%s\n", output.Version)
	return err
}

func renderLedgerStats(cmd *cobra.Command, output ledgercmd.ReadStatsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Transactions\t%s\n", output.Transactions); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Accounts\t%d\n", output.Accounts)
	return err
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

func renderLedgerTransaction(cmd *cobra.Command, output ledgercmd.GetTransactionOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	reference := ""
	if output.Transaction.Reference != nil {
		reference = *output.Transaction.Reference
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", output.Transaction.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Reference\t%s\n", reference); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Timestamp\t%s\n", output.Transaction.Timestamp.Format(time.RFC3339))
	return err
}

func renderLedgerVolumes(cmd *cobra.Command, output ledgercmd.ListVolumesOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Volumes) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No volumes found.")
		return err
	}
	for _, volume := range output.Volumes {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n",
			volume.Account, volume.Asset, volume.Input, volume.Output, volume.Balance); err != nil {
			return err
		}
	}
	return nil
}

func parseMetadataFlags(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	ret := map[string]string{}
	for _, value := range values {
		if value == "" {
			continue
		}
		key, val, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("metadata must use key=value format")
		}
		ret[key] = val
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}

func parseOptionalRFC3339(value string, name string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}
	return &parsed, nil
}
