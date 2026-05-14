package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	ledgercmd "github.com/formancehq/fctl/v4/internal/commands/ledger"
	v4prompt "github.com/formancehq/fctl/v4/internal/prompt"
	v4render "github.com/formancehq/fctl/v4/internal/render"
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
	command.AddCommand(newLedgerSchemasCommand())
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
	command.AddCommand(newLedgerAccountsQueryCommand())
	command.AddCommand(newLedgerAccountsSetMetadataCommand())
	command.AddCommand(newLedgerAccountsShowCommand())
	return command
}

func newLedgerSchemasCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "schemas",
		Short: "Manage ledger schemas",
	}
	command.AddCommand(newLedgerSchemasListCommand())
	command.AddCommand(newLedgerSchemasShowCommand())
	command.AddCommand(newLedgerSchemasInsertCommand())
	return command
}

func newLedgerSchemasListCommand() *cobra.Command {
	var ledger string
	var pageSize int64 = 15
	var cursor string
	var apiVersion string

	command := &cobra.Command{
		Use:   "list",
		Short: "List ledger schemas",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
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
			service := ledgercmd.ListSchemasService{
				Handlers: ledgercmd.SDKListSchemasHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureListSchemas,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.ListSchemasInput{
				Ledger:   ledger,
				PageSize: pageSize,
				Cursor:   cursor,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerSchemas(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerSchemasShowCommand() *cobra.Command {
	var ledger string
	var apiVersion string

	command := &cobra.Command{
		Use:   "show <version>",
		Short: "Show a ledger schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
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
			service := ledgercmd.GetSchemaService{
				Handlers: ledgercmd.SDKGetSchemaHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureGetSchema,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.GetSchemaInput{
				Ledger:  ledger,
				Version: args[0],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerSchema(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

	return command
}

func newLedgerSchemasInsertCommand() *cobra.Command {
	var ledger string
	var file string
	var confirm bool
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:   "insert <version>",
		Short: "Insert a ledger schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger schemas insert requires --confirm")
			}
			if file == "" {
				return fmt.Errorf("ledger schemas insert requires --file <path>|-")
			}

			var data []byte
			var err error
			if file == "-" {
				data, err = io.ReadAll(cmd.InOrStdin())
			} else {
				data, err = os.ReadFile(file)
			}
			if err != nil {
				return err
			}

			rt, err := stackRuntimeFromCommand(cmd)
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
			service := ledgercmd.InsertSchemaService{
				Handlers: ledgercmd.SDKInsertSchemaHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureInsertSchema,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.InsertSchemaInput{
				Ledger:         ledger,
				Version:        args[0],
				Data:           data,
				IdempotencyKey: idempotencyKey,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerSchemaInserted(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&file, "file", "", "Read schema JSON from file, or - for stdin")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the schema insertion")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

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

			rt, err := stackRuntimeFromCommand(cmd)
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

func newLedgerAccountsQueryCommand() *cobra.Command {
	var ledger string
	var schemaVersion string
	var pageSize int64
	var cursor string
	var expand string
	var pit string
	var reverse bool
	var sort string
	var variables []string
	var apiVersion string

	command := &cobra.Command{
		Use:   "query <query-id>",
		Short: "Run a ledger account query template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parsedVars, err := parseKeyValueFlags(variables, "var")
			if err != nil {
				return err
			}
			parsedPit, err := parseOptionalRFC3339(pit, "pit")
			if err != nil {
				return err
			}

			rt, err := stackRuntimeFromCommand(cmd)
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
			service := ledgercmd.RunAccountQueryService{
				Handlers: ledgercmd.SDKRunAccountQueryHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         ledgercmd.ProductLedger,
						Feature:         ledgercmd.FeatureRunQuery,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), ledgercmd.RunAccountQueryInput{
				Ledger:        ledger,
				QueryID:       args[0],
				SchemaVersion: schemaVersion,
				PageSize:      pageSize,
				Cursor:        cursor,
				Expand:        expand,
				Pit:           parsedPit,
				Reverse:       reverse,
				Sort:          sort,
				Vars:          parsedVars,
			})
			if err != nil {
				return err
			}

			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerAccountQuery(cmd, output)
		},
	}

	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&schemaVersion, "schema-version", "", "Ledger schema version")
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&expand, "expand", "", "Expand response fields")
	command.Flags().StringVar(&pit, "pit", "", "Point-in-time timestamp (RFC3339)")
	command.Flags().BoolVar(&reverse, "reverse", false, "Reverse result order")
	command.Flags().StringVar(&sort, "sort", "", "Sort results as field:asc or field:desc")
	command.Flags().StringArrayVar(&variables, "var", nil, "Query template variable as key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")

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
			rt, err := stackRuntimeFromCommand(cmd)
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

			rt, err := stackRuntimeFromCommand(cmd)
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

			rt, err := stackRuntimeFromCommand(cmd)
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
			rt, err := stackRuntimeFromCommand(cmd)
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

			rt, err := stackRuntimeFromCommand(cmd)
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
	var metadataFile string
	var apiVersion string

	command := &cobra.Command{
		Use:     "set-metadata <ledger> [key=value]...",
		Aliases: []string{"sm", "set-meta"},
		Short:   "Set metadata on a ledger",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("ledger set-metadata requires --confirm")
			}

			metadata, err := parseMetadataFile(cmd, metadataFile)
			if err != nil {
				return err
			}
			argMetadata, err := parseMetadataFlags(args[1:])
			if err != nil {
				return err
			}
			for key, value := range argMetadata {
				if metadata == nil {
					metadata = map[string]string{}
				}
				metadata[key] = value
			}

			rt, err := stackRuntimeFromCommand(cmd)
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
	command.Flags().StringVar(&metadataFile, "metadata-file", "", "Read metadata JSON object from path or stdin with -")
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

			rt, err := stackRuntimeFromCommand(cmd)
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
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ledgerName, err := resolveLedgerCreateName(cmd, args)
			if err != nil {
				return err
			}
			if !confirm {
				approved, err := confirmLedgerCreate(cmd, ledgerName)
				if err != nil {
					return err
				}
				if !approved {
					return v4prompt.ErrCancelled
				}
			}

			parsedFeatures, err := parseKeyValueFlags(features, "feature")
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}

			rt, err := stackRuntimeFromCommand(cmd)
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
				Name:     ledgerName,
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

func resolveLedgerCreateName(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0]), nil
	}
	nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag)
	if err != nil {
		return "", err
	}
	wizard := v4prompt.NewWizardWithColor(cmd.InOrStdin(), cmd.ErrOrStderr(), commandColorEnabled(cmd))
	if nonInteractive || !wizard.Available() {
		return "", fmt.Errorf("ledger create requires <name>")
	}
	value, err := wizard.Input("Ledger name", "default", false)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("ledger name is required")
	}
	fmt.Fprintln(cmd.OutOrStdout(), styledKeyValueLine(cmd, "Name", value))
	return value, nil
}

func confirmLedgerCreate(cmd *cobra.Command, ledgerName string) (bool, error) {
	nonInteractive, err := cmd.Root().PersistentFlags().GetBool(nonInteractiveFlag)
	if err != nil {
		return false, err
	}
	wizard := v4prompt.NewWizardWithColor(cmd.InOrStdin(), cmd.ErrOrStderr(), commandColorEnabled(cmd))
	if nonInteractive || !wizard.Available() {
		return false, fmt.Errorf("ledger create requires --confirm")
	}
	return wizard.Confirm(fmt.Sprintf("Create ledger %s?", ledgerName), "Create", "Cancel")
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

			rt, err := stackRuntimeFromCommand(cmd)
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
			rt, err := stackRuntimeFromCommand(cmd)
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
			rt, err := stackRuntimeFromCommand(cmd)
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
	command.AddCommand(newLedgerTransactionsExplainCommand())
	command.AddCommand(newLedgerTransactionsListCommand())
	command.AddCommand(newLedgerTransactionsRevertCommand())
	command.AddCommand(newLedgerTransactionsRunScriptCommand("run-script", nil, false))
	command.AddCommand(newLedgerTransactionsRunScriptCommand("num", nil, true))
	command.AddCommand(newLedgerTransactionsSendCommand(false))
	command.AddCommand(newLedgerTransactionsSetMetadataCommand())
	command.AddCommand(newLedgerTransactionsShowCommand())
	return command
}

func newLedgerTransactionsExplainCommand() *cobra.Command {
	var ledger string
	var apiVersion string

	command := &cobra.Command{
		Use:   "explain <transaction-id>",
		Short: "Explain a ledger transaction (requires ledger API v3+)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			if ledger == "" {
				ledger = rt.Context.Defaults["ledger"]
			}
			if ledger == "" {
				ledger = "default"
			}

			request := capabilities.VersionResolutionRequest{
				Product:         ledgercmd.ProductLedger,
				Feature:         ledgercmd.FeatureExplainTransaction,
				HandlerVersions: []capabilities.APIVersion{"v3"},
			}
			if apiVersion != "" {
				request.Policy = capabilities.VersionPolicyPinned
				request.PinnedVersion = capabilities.APIVersion(apiVersion)
			}
			selected, err := rt.ResolveAPIVersion(cmd.Context(), request)
			if err != nil {
				var unsupported *capabilities.UnsupportedFeatureError
				if errors.As(err, &unsupported) {
					return fmt.Errorf("ledger transactions explain requires ledger API v3+; target ledger component %s supports %s",
						unsupported.ComponentVersion,
						strings.Join(apiVersionsToStrings(unsupported.Supported), ","),
					)
				}
				return err
			}
			return fmt.Errorf("ledger transactions explain resolved %s for ledger %s transaction %s, but explainTransaction is not exposed by the current stack v3.2.4 spec or formance-sdk-go yet",
				selected,
				ledger,
				args[0],
			)
		},
	}
	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")
	return command
}

func newLedgerTransactionsRunScriptCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var ledger string
	var file string
	var accountVars []string
	var amountVars []string
	var portionVars []string
	var metadata []string
	var timestamp string
	var reference string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Execute a Numscript program on a ledger",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command ledger transactions num has been deprecated, use ledger transactions run-script --file <path>|-")
			}
			if file != "" && len(args) == 1 {
				return fmt.Errorf("positional file and --file are mutually exclusive")
			}
			if file == "" && len(args) == 1 {
				file = args[0]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use ledger transactions run-script --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("ledger transactions run-script requires --file <path>|-")
			}
			if !confirm {
				return fmt.Errorf("ledger transactions run-script requires --confirm")
			}

			script, err := readLedgerCommandFile(cmd, file)
			if err != nil {
				return err
			}
			parsedAccountVars, err := parseKeyValueFlags(accountVars, "account-var")
			if err != nil {
				return err
			}
			parsedAmountVars, err := parseKeyValueFlags(amountVars, "amount-var")
			if err != nil {
				return err
			}
			parsedPortionVars, err := parseKeyValueFlags(portionVars, "portion-var")
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			parsedTimestamp, err := parseOptionalRFC3339(timestamp, "timestamp")
			if err != nil {
				return err
			}

			rt, err := stackRuntimeFromCommand(cmd)
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
			service := ledgercmd.RunScriptService{
				Handlers: ledgercmd.SDKRunScriptHandlers(sdk),
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
			output, err := service.Run(cmd.Context(), ledgercmd.RunScriptInput{
				Ledger:      ledger,
				Script:      string(script),
				AccountVars: parsedAccountVars,
				AmountVars:  parsedAmountVars,
				PortionVars: parsedPortionVars,
				Metadata:    parsedMetadata,
				Reference:   reference,
				Timestamp:   parsedTimestamp,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderLedgerSentTransaction(cmd, ledgercmd.SendTransactionOutput{
				APIVersion:  output.APIVersion,
				Transaction: output.Transaction,
			})
		},
	}
	if deprecated {
		command.Deprecated = "use ledger transactions run-script --file <path>|-"
	}
	command.Flags().StringVar(&ledger, "ledger", "", "Ledger name")
	command.Flags().StringVar(&file, "file", "", "Path to Numscript file or - for stdin")
	command.Flags().StringSliceVar(&accountVars, "account-var", nil, "Numscript account variable as key=value")
	command.Flags().StringSliceVar(&amountVars, "amount-var", nil, "Numscript amount variable as key=amount/asset")
	command.Flags().StringSliceVar(&portionVars, "portion-var", nil, "Numscript portion variable as key=value")
	command.Flags().StringSliceVar(&metadata, "metadata", nil, "Metadata key=value to apply")
	command.Flags().StringVar(&timestamp, "timestamp", "", "Transaction timestamp (RFC3339)")
	command.Flags().StringVar(&reference, "reference", "", "Transaction reference")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm the transaction creation")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")
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

	argsValidator := cobra.NoArgs
	if deprecatedRootAlias {
		argsValidator = func(_ *cobra.Command, args []string) error {
			if len(args) == 0 || len(args) == 3 || len(args) == 4 {
				return nil
			}
			return fmt.Errorf("ledger send accepts either flags or [source] <destination> <amount> <asset>")
		}
	}

	command := &cobra.Command{
		Use:   "send",
		Short: "Send from one ledger account to another",
		Args:  argsValidator,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecatedRootAlias {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command ledger send has been deprecated, use ledger transactions send")
			}
			if deprecatedRootAlias && len(args) > 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional ledger send arguments have been deprecated, use ledger transactions send with explicit flags")
				if len(args) == 3 {
					source = "world"
					destination = args[0]
					amount = args[1]
					asset = args[2]
				} else {
					source = args[0]
					destination = args[1]
					amount = args[2]
					asset = args[3]
				}
			}
			if !confirm {
				return fmt.Errorf("ledger transactions send requires --confirm")
			}

			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}

			rt, err := stackRuntimeFromCommand(cmd)
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

			rt, err := stackRuntimeFromCommand(cmd)
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
			rt, err := stackRuntimeFromCommand(cmd)
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
	var metadata []string
	var start string
	var end string
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
			if cmd.Flags().Changed("start-time") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --start-time has been deprecated, use --start")
			}
			if cmd.Flags().Changed("end-time") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --end-time has been deprecated, use --end")
			}

			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			parsedStart, err := parseOptionalRFC3339(start, "start")
			if err != nil {
				return err
			}
			parsedEnd, err := parseOptionalRFC3339(end, "end")
			if err != nil {
				return err
			}

			rt, err := stackRuntimeFromCommand(cmd)
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
				Metadata:    parsedMetadata,
				StartTime:   parsedStart,
				EndTime:     parsedEnd,
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
	command.Flags().StringSliceVar(&metadata, "metadata", nil, "Filter by metadata key=value")
	command.Flags().StringVar(&start, "start", "", "Filter transactions after this RFC3339 timestamp")
	command.Flags().StringVar(&start, "start-time", "", "Deprecated alias for --start")
	command.Flags().StringVar(&end, "end", "", "Filter transactions before this RFC3339 timestamp")
	command.Flags().StringVar(&end, "end-time", "", "Deprecated alias for --end")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin ledger API version")
	_ = command.Flags().MarkDeprecated("src", "use --source")
	_ = command.Flags().MarkDeprecated("dst", "use --destination")
	_ = command.Flags().MarkDeprecated("start-time", "use --start")
	_ = command.Flags().MarkDeprecated("end-time", "use --end")

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

			rt, err := stackRuntimeFromCommand(cmd)
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

			rt, err := stackRuntimeFromCommand(cmd)
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

			rt, err := stackRuntimeFromCommand(cmd)
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

			rt, err := stackRuntimeFromCommand(cmd)
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
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Ledgers) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No ledgers found."))
		return err
	}
	rows := make([][]string, 0, len(output.Ledgers))
	for _, ledger := range output.Ledgers {
		rows = append(rows, []string{
			ledger.Name,
			ledger.Bucket,
			ledger.AddedAt.Format(time.RFC3339),
		})
	}
	return v4render.Table(cmd.OutOrStdout(), []string{"Name", "Bucket", "Created at"}, rows)
}

func renderLedgerAccounts(cmd *cobra.Command, output ledgercmd.ListAccountsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Accounts) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No accounts found."))
		return err
	}
	rows := make([][]string, 0, len(output.Accounts))
	for _, account := range output.Accounts {
		rows = append(rows, []string{account.Address})
	}
	return v4render.Table(cmd.OutOrStdout(), []string{"Address"}, rows)
}

func renderLedgerAccount(cmd *cobra.Command, output ledgercmd.GetAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if err := v4render.KeyValues(cmd.OutOrStdout(), [][]string{{"Address", output.Account.Address}}); err != nil {
		return err
	}
	if len(output.Account.Volumes) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No volumes."))
		return err
	}
	rows := make([][]string, 0, len(output.Account.Volumes))
	for asset, volume := range output.Account.Volumes {
		rows = append(rows, []string{asset, volume.Input, volume.Output, volume.Balance})
	}
	return v4render.Table(cmd.OutOrStdout(), []string{"Asset", "Input", "Output", "Balance"}, rows)
}

func renderLedgerAccountQuery(cmd *cobra.Command, output ledgercmd.RunAccountQueryOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if err := v4render.KeyValues(cmd.OutOrStdout(), [][]string{{"Query", output.QueryID}}); err != nil {
		return err
	}
	if len(output.Accounts) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No accounts found."))
		return err
	}
	rows := make([][]string, 0, len(output.Accounts))
	for _, account := range output.Accounts {
		rows = append(rows, []string{account.Address})
	}
	if err := v4render.Table(cmd.OutOrStdout(), []string{"Address"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderLedgerAccountMetadataSet(cmd *cobra.Command, output ledgercmd.AddAccountMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata added.")
	return err
}

func renderLedgerAccountMetadataDeleted(cmd *cobra.Command, output ledgercmd.DeleteAccountMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata deleted.")
	return err
}

func renderLedgerCreated(cmd *cobra.Command, output ledgercmd.CreateLedgerOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger %s created.\n", output.Name)
	return err
}

func renderLedgerExported(cmd *cobra.Command, output ledgercmd.ExportLogsOutput, file string) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger %s exported to %s.\n", output.Ledger, file)
	return err
}

func renderLedgerImported(cmd *cobra.Command, output ledgercmd.ImportLogsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger %s imported.\n", output.Ledger)
	return err
}

func renderLedgerSchemas(cmd *cobra.Command, output ledgercmd.ListSchemasOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Schemas) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No schemas found."))
		return err
	}
	rows := make([][]string, 0, len(output.Schemas))
	for _, schema := range output.Schemas {
		rows = append(rows, []string{
			schema.Version,
			schema.CreatedAt.Format(time.RFC3339),
			fmt.Sprintf("%d", schema.ChartSegments),
			fmt.Sprintf("%d", schema.QueryTemplates),
			fmt.Sprintf("%d", schema.TransactionModels),
		})
	}
	if err := v4render.Table(cmd.OutOrStdout(), []string{"Version", "Created at", "Chart segments", "Query templates", "Transaction models"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderLedgerSchema(cmd *cobra.Command, output ledgercmd.GetSchemaOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	return v4render.KeyValues(cmd.OutOrStdout(), [][]string{
		{"Version", output.Schema.Version},
		{"Created at", output.Schema.CreatedAt.Format(time.RFC3339)},
		{"Chart segments", fmt.Sprintf("%d", len(output.Schema.Chart))},
		{"Query templates", fmt.Sprintf("%d", len(output.Schema.Queries))},
		{"Transaction models", fmt.Sprintf("%d", len(output.Schema.Transactions))},
	})
}

func renderLedgerSchemaInserted(cmd *cobra.Command, output ledgercmd.InsertSchemaOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Schema %s inserted in ledger %s.\n", output.Version, output.Ledger)
	return err
}

func renderLedgerMetadataSet(cmd *cobra.Command, output ledgercmd.UpdateLedgerMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata added.")
	return err
}

func renderLedgerMetadataDeleted(cmd *cobra.Command, output ledgercmd.DeleteLedgerMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata deleted.")
	return err
}

func renderLedgerInfo(cmd *cobra.Command, output ledgercmd.ReadInfoOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	return v4render.KeyValues(cmd.OutOrStdout(), [][]string{
		{"Server", output.Server},
		{"Version", output.Version},
	})
}

func renderLedgerStats(cmd *cobra.Command, output ledgercmd.ReadStatsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	return v4render.KeyValues(cmd.OutOrStdout(), [][]string{
		{"Transactions", output.Transactions},
		{"Accounts", fmt.Sprintf("%d", output.Accounts)},
	})
}

func renderLedgerTransactions(cmd *cobra.Command, output ledgercmd.ListTransactionsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Transactions) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No transactions found."))
		return err
	}
	rows := make([][]string, 0, len(output.Transactions))
	for _, transaction := range output.Transactions {
		reference := ""
		if transaction.Reference != nil {
			reference = *transaction.Reference
		}
		rows = append(rows, []string{
			transaction.ID,
			reference,
			transaction.Timestamp.Format(time.RFC3339),
		})
	}
	return v4render.Table(cmd.OutOrStdout(), []string{"ID", "Reference", "Timestamp"}, rows)
}

func renderLedgerTransactionsCount(cmd *cobra.Command, output ledgercmd.CountTransactionsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	return v4render.KeyValues(cmd.OutOrStdout(), [][]string{{"Count", fmt.Sprintf("%d", output.Count)}})
}

func renderLedgerSentTransaction(cmd *cobra.Command, output ledgercmd.SendTransactionOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	reference := ""
	if output.Transaction.Reference != nil {
		reference = *output.Transaction.Reference
	}
	return renderLedgerTransactionSummary(cmd, output.Transaction.ID, reference, output.Transaction.Timestamp)
}

func renderLedgerTransactionMetadataSet(cmd *cobra.Command, output ledgercmd.AddTransactionMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata added.")
	return err
}

func renderLedgerTransactionMetadataDeleted(cmd *cobra.Command, output ledgercmd.DeleteTransactionMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Metadata deleted.")
	return err
}

func renderLedgerTransaction(cmd *cobra.Command, output ledgercmd.GetTransactionOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	reference := ""
	if output.Transaction.Reference != nil {
		reference = *output.Transaction.Reference
	}
	return renderLedgerTransactionSummary(cmd, output.Transaction.ID, reference, output.Transaction.Timestamp)
}

func renderLedgerRevertedTransaction(cmd *cobra.Command, output ledgercmd.RevertTransactionOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	reference := ""
	if output.Transaction.Reference != nil {
		reference = *output.Transaction.Reference
	}
	return renderLedgerTransactionSummary(cmd, output.Transaction.ID, reference, output.Transaction.Timestamp)
}

func renderLedgerVolumes(cmd *cobra.Command, output ledgercmd.ListVolumesOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Volumes) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No volumes found."))
		return err
	}
	rows := make([][]string, 0, len(output.Volumes))
	for _, volume := range output.Volumes {
		rows = append(rows, []string{volume.Account, volume.Asset, volume.Input, volume.Output, volume.Balance})
	}
	return v4render.Table(cmd.OutOrStdout(), []string{"Account", "Asset", "Input", "Output", "Balance"}, rows)
}

func renderLedgerTransactionSummary(cmd *cobra.Command, id string, reference string, timestamp time.Time) error {
	return v4render.KeyValues(cmd.OutOrStdout(), [][]string{
		{"ID", id},
		{"Reference", reference},
		{"Timestamp", timestamp.Format(time.RFC3339)},
	})
}

func parseMetadataFlags(values []string) (map[string]string, error) {
	return parseKeyValueFlags(values, "metadata")
}

func parseMetadataFile(cmd *cobra.Command, file string) (map[string]string, error) {
	if file == "" {
		return nil, nil
	}
	data, err := readLedgerCommandFile(cmd, file)
	if err != nil {
		return nil, err
	}
	var metadata map[string]string
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parse metadata file: %w", err)
	}
	if len(metadata) == 0 {
		return nil, nil
	}
	return metadata, nil
}

func readLedgerCommandFile(cmd *cobra.Command, file string) ([]byte, error) {
	if file == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(file)
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
