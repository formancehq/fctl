package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
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
	command.AddCommand(newLedgerCreateCommand())
	command.AddCommand(newLedgerDeleteMetadataCommand())
	command.AddCommand(newLedgerExportCommand())
	command.AddCommand(newLedgerImportCommand())
	command.AddCommand(newLedgerListCommand())
	command.AddCommand(newLedgerSetMetadataCommand())
	command.AddCommand(newLedgerTransactionsSendCommand(true))
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
	command.AddCommand(newLedgerAccountsDeleteMetadataCommand())
	command.AddCommand(newLedgerAccountsSetMetadataCommand())
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

func newLedgerAccountsSetMetadataCommand() *cobra.Command {
	var ledger string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "set-metadata <account> <key=value>...",
		Aliases: []string{"sm", "set-meta"},
		Short:   "Set metadata on a ledger account",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger accounts set-metadata requires --confirm")
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

			metadata, err := parseMetadataFlags(args[1:])
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
			service := ledgercmd.AddAccountMetadataService{
				Handlers: ledgercmd.SDKAddAccountMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureAddAccountMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.AddAccountMetadataInput{
				Ledger:   ledger,
				Account:  args[0],
				Metadata: metadata,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerAccountMetadataSet(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the metadata update")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerAccountsDeleteMetadataCommand() *cobra.Command {
	var ledger string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "delete-metadata <account> <key>",
		Aliases: []string{"dm", "del-meta"},
		Short:   "Delete metadata from a ledger account",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger accounts delete-metadata requires --confirm")
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
			service := ledgercmd.DeleteAccountMetadataService{
				Handlers: ledgercmd.SDKDeleteAccountMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureDeleteAccountMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.DeleteAccountMetadataInput{
				Ledger:  ledger,
				Account: args[0],
				Key:     args[1],
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerAccountMetadataDeleted(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the metadata deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerExportCommand() *cobra.Command {
	var ledger string
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:   "export",
		Short: "Export ledger logs",
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
			service := ledgercmd.ExportLogsService{
				Handlers: ledgercmd.SDKExportLogsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureExportLogs,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.ExportLogsInput{
				Ledger: ledger,
			})
			if err != nil {
				return err
			}

			if file != "" && file != "-" {
				if err := os.WriteFile(file, output.Data, 0o600); err != nil {
					return err
				}
				return renderLedgerExported(cmd, output, file)
			}
			_, err = cmd.OutOrStdout().Write(output.Data)
			return err
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&file, "file", "", "Write export to file, or - for stdout")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerImportCommand() *cobra.Command {
	var file string
	var input string
	var resume bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "import <ledger> [file]",
		Short: "Import ledger logs",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if input != "" {
				if file != "" {
					return fmt.Errorf("--input and --file are mutually exclusive")
				}
				file = input
			}
			if len(args) == 2 {
				if file != "" {
					return fmt.Errorf("positional file and --file are mutually exclusive")
				}
				file = args[1]
			}
			if file == "" {
				return fmt.Errorf("ledger import requires --file <path>|-")
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
			service := ledgercmd.ImportLogsService{
				Handlers: ledgercmd.SDKImportLogsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureImportLogs,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}

			importInput := ledgercmd.ImportLogsInput{
				Ledger:            args[0],
				FilePath:          file,
				ResumeFromLastLog: resume,
			}
			if file == "-" {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				importInput.FilePath = ""
				importInput.Data = data
			}

			output, err := service.Run(cmd.Context(), importInput)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerImported(cmd, output)
		},
	}

	command.Flags().StringVar(&file, "file", "", "Read import from file, or - for stdin")
	command.Flags().StringVar(&input, "input", "", "Read import from file, or - for stdin")
	_ = command.Flags().MarkDeprecated("input", "use --file")
	command.Flags().BoolVar(&resume, "resume-from-last-log", false, "Resume import after the latest log already present in the ledger")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerSetMetadataCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "set-metadata <ledger> <key=value>...",
		Aliases: []string{"sm", "set-meta"},
		Short:   "Set metadata on a ledger",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger set-metadata requires --confirm")
			}

			metadata, err := parseMetadataFlags(args[1:])
			if err != nil {
				return err
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
			service := ledgercmd.UpdateLedgerMetadataService{
				Handlers: ledgercmd.SDKUpdateLedgerMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureUpdateLedgerMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.UpdateLedgerMetadataInput{
				Ledger:   args[0],
				Metadata: metadata,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerMetadataSet(cmd, output)
		},
	}

	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the metadata update")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerDeleteMetadataCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "delete-metadata <ledger> <key>",
		Aliases: []string{"dm", "del-meta"},
		Short:   "Delete metadata from a ledger",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger delete-metadata requires --confirm")
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
			service := ledgercmd.DeleteLedgerMetadataService{
				Handlers: ledgercmd.SDKDeleteLedgerMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureDeleteLedgerMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.DeleteLedgerMetadataInput{
				Ledger: args[0],
				Key:    args[1],
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerMetadataDeleted(cmd, output)
		},
	}

	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the metadata deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerCreateCommand() *cobra.Command {
	var bucket string
	var features []string
	var metadata []string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"c", "cr"},
		Short:   "Create a ledger",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger create requires --confirm")
			}

			parsedFeatures, err := parseKeyValueFlags(features, "feature")
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
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
			service := ledgercmd.CreateLedgerService{
				Handlers: ledgercmd.SDKCreateLedgerHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureCreateLedger,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.CreateLedgerInput{
				Name:     args[0],
				Bucket:   bucket,
				Features: parsedFeatures,
				Metadata: parsedMetadata,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerCreated(cmd, output)
		},
	}

	command.Flags().StringVar(&bucket, "bucket", "", "Bucket in which to create the ledger")
	command.Flags().StringSliceVar(&features, "feature", nil, "Feature key=value to enable")
	command.Flags().StringSliceVar(&metadata, "metadata", nil, "Metadata key=value to apply")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the ledger creation")
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
	command.AddCommand(newLedgerTransactionsCountCommand())
	command.AddCommand(newLedgerTransactionsDeleteMetadataCommand())
	command.AddCommand(newLedgerTransactionsListCommand())
	command.AddCommand(newLedgerTransactionsRevertCommand())
	command.AddCommand(newLedgerTransactionsSendCommand(false))
	command.AddCommand(newLedgerTransactionsSetMetadataCommand())
	command.AddCommand(newLedgerTransactionsShowCommand())
	return command
}

func newLedgerTransactionsSendCommand(deprecatedRootAlias bool) *cobra.Command {
	var ledger string
	var source string
	var destination string
	var amount string
	var asset string
	var reference string
	var metadata []string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "send",
		Short: "Send from one ledger account to another",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deprecatedRootAlias {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command ledger send has been deprecated, use ledger transactions send")
			}
			if !confirm {
				return fmt.Errorf("ledger transactions send requires --confirm")
			}

			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
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
			service := ledgercmd.SendTransactionService{
				Handlers: ledgercmd.SDKSendTransactionHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureCreateTransaction,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.SendTransactionInput{
				Ledger:      ledger,
				Source:      source,
				Destination: destination,
				Amount:      amount,
				Asset:       asset,
				Reference:   reference,
				Metadata:    parsedMetadata,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerSentTransaction(cmd, output)
		},
	}
	if deprecatedRootAlias {
		command.Aliases = []string{"s"}
		command.Deprecated = "use ledger transactions send"
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&source, "source", "", "Source account")
	command.Flags().StringVar(&destination, "destination", "", "Destination account")
	command.Flags().StringVar(&amount, "amount", "", "Amount to send")
	command.Flags().StringVar(&asset, "asset", "", "Asset to send")
	command.Flags().StringVar(&reference, "reference", "", "Transaction reference")
	command.Flags().StringSliceVar(&metadata, "metadata", nil, "Metadata key=value to apply")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the transaction creation")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

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

func newLedgerTransactionsCountCommand() *cobra.Command {
	var ledger string
	var account string
	var source string
	var destination string
	var reference string
	var apiVersion string

	command := &cobra.Command{
		Use:   "count",
		Short: "Count ledger transactions",
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
			service := ledgercmd.CountTransactionsService{
				Handlers: ledgercmd.SDKCountTransactionsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureCountTransactions,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.CountTransactionsInput{
				Ledger:      ledger,
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
			return renderLedgerTransactionsCount(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
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

func newLedgerTransactionsSetMetadataCommand() *cobra.Command {
	var ledger string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "set-metadata <transaction-id> <key=value>...",
		Aliases: []string{"sm", "set-meta"},
		Short:   "Set metadata on a ledger transaction",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger transactions set-metadata requires --confirm")
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

			metadata, err := parseMetadataFlags(args[1:])
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
			service := ledgercmd.AddTransactionMetadataService{
				Handlers: ledgercmd.SDKAddTransactionMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureAddTransactionMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.AddTransactionMetadataInput{
				Ledger:        ledger,
				TransactionID: args[0],
				Metadata:      metadata,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerTransactionMetadataSet(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the metadata update")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerTransactionsDeleteMetadataCommand() *cobra.Command {
	var ledger string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "delete-metadata <transaction-id> <key>",
		Aliases: []string{"dm", "del-meta"},
		Short:   "Delete metadata from a ledger transaction",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger transactions delete-metadata requires --confirm")
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
			service := ledgercmd.DeleteTransactionMetadataService{
				Handlers: ledgercmd.SDKDeleteTransactionMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureDeleteTransactionMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.DeleteTransactionMetadataInput{
				Ledger:        ledger,
				TransactionID: args[0],
				Key:           args[1],
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerTransactionMetadataDeleted(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the metadata deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerTransactionsRevertCommand() *cobra.Command {
	var ledger string
	var atEffectiveDate bool
	var force bool
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "revert <transaction-id>",
		Short: "Revert a ledger transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger transactions revert requires --confirm")
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
			service := ledgercmd.RevertTransactionService{
				Handlers: ledgercmd.SDKRevertTransactionHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureRevertTransaction,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.RevertTransactionInput{
				Ledger:          ledger,
				TransactionID:   args[0],
				AtEffectiveDate: atEffectiveDate,
				Force:           force,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerRevertedTransaction(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().BoolVar(&atEffectiveDate, "at-effective-date", false, "Revert at the original transaction effective date")
	command.Flags().BoolVar(&force, "force", false, "Force revert and bypass ledger checks when supported")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the transaction revert")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

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

func renderLedgerAccountMetadataSet(cmd *cobra.Command, output ledgercmd.AddAccountMetadataOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata added.")
	return err
}

func renderLedgerAccountMetadataDeleted(cmd *cobra.Command, output ledgercmd.DeleteAccountMetadataOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata deleted.")
	return err
}

func renderLedgerCreated(cmd *cobra.Command, output ledgercmd.CreateLedgerOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger %s created.\n", output.Name)
	return err
}

func renderLedgerExported(cmd *cobra.Command, output ledgercmd.ExportLogsOutput, file string) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger %s exported to %s.\n", output.Ledger, file)
	return err
}

func renderLedgerImported(cmd *cobra.Command, output ledgercmd.ImportLogsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger %s imported.\n", output.Ledger)
	return err
}

func renderLedgerMetadataSet(cmd *cobra.Command, output ledgercmd.UpdateLedgerMetadataOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata added.")
	return err
}

func renderLedgerMetadataDeleted(cmd *cobra.Command, output ledgercmd.DeleteLedgerMetadataOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata deleted.")
	return err
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

func renderLedgerTransactionsCount(cmd *cobra.Command, output ledgercmd.CountTransactionsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Count\t%d\n", output.Count)
	return err
}

func renderLedgerSentTransaction(cmd *cobra.Command, output ledgercmd.SendTransactionOutput) error {
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

func renderLedgerTransactionMetadataSet(cmd *cobra.Command, output ledgercmd.AddTransactionMetadataOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata added.")
	return err
}

func renderLedgerTransactionMetadataDeleted(cmd *cobra.Command, output ledgercmd.DeleteTransactionMetadataOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata deleted.")
	return err
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

func renderLedgerRevertedTransaction(cmd *cobra.Command, output ledgercmd.RevertTransactionOutput) error {
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
	return parseKeyValueFlags(values, "metadata")
}

func parseKeyValueFlags(values []string, name string) (map[string]string, error) {
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
			return nil, fmt.Errorf("%s must use key=value format", name)
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
