package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	paymentscmd "github.com/formancehq/fctl/v4/internal/commands/payments"
)

func newPaymentsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "payments",
		Short: "Manage payments",
	}
	command.AddCommand(newPaymentsAccountsCommand())
	command.AddCommand(newPaymentsBankAccountsCommand("bank-accounts", nil, false))
	command.AddCommand(newPaymentsBankAccountsCommand("bank_accounts", []string{"bacc", "ba", "bac", "baccount"}, true))
	command.AddCommand(newPaymentsConnectorsCommand())
	command.AddCommand(newPaymentsPaymentsCommand())
	command.AddCommand(newPaymentsPoolsCommand())
	command.AddCommand(newPaymentsTasksCommand())
	command.AddCommand(newPaymentsTransferInitiationCommand("transfer-initiation", []string{"ti"}, false))
	command.AddCommand(newPaymentsTransferInitiationCommand("transfer_initiation", nil, true))
	command.AddCommand(newPaymentsVersionsCommand())
	return command
}

func newPaymentsVersionsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "Show payments component and API versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			versions, err := rt.ComponentVersions(cmd.Context())
			if err != nil {
				return err
			}
			for _, version := range versions {
				if version.Product != paymentscmd.ProductPayments {
					continue
				}
				apiVersions, _ := rt.Compatibility.APIVersionsFor(version.Product, version.Version)
				output := paymentsVersionsOutput{
					Name:        string(version.Product),
					Version:     version.Version,
					Health:      version.Health,
					APIVersions: apiVersionsToStrings(apiVersions),
					APIPolicy:   string(rt.APIPolicyFor(version.Product)),
				}
				if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
					return err
				}
				health := "unhealthy"
				if output.Health {
					health = "healthy"
				}
				if terminalOutputEnabled(cmd) {
					return writeStyledKeyValues(cmd,
						styledKeyValue{Label: "Component", Value: output.Name},
						styledKeyValue{Label: "Version", Value: output.Version},
						styledKeyValue{Label: "Health", Value: health},
						styledKeyValue{Label: "API", Value: fmt.Sprintf("%v", output.APIVersions)},
						styledKeyValue{Label: "Policy", Value: output.APIPolicy},
					)
				}
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s api=%v policy=%s\n",
					output.Name, output.Version, health, output.APIVersions, output.APIPolicy)
				return err
			}
			return fmt.Errorf("payments component not found in target versions")
		},
	}
}

type paymentsVersionsOutput struct {
	Name        string   `json:"name" yaml:"name"`
	Version     string   `json:"version" yaml:"version"`
	Health      bool     `json:"health" yaml:"health"`
	APIVersions []string `json:"apiVersions" yaml:"apiVersions"`
	APIPolicy   string   `json:"apiPolicy" yaml:"apiPolicy"`
}

func newPaymentsConnectorsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "connectors",
		Aliases: []string{"connector", "co"},
		Short:   "Manage payment connectors",
	}
	command.AddCommand(newPaymentsConnectorsInstallCommand())
	command.AddCommand(newPaymentsConnectorsListCommand())
	command.AddCommand(newPaymentsConnectorsConfigCommand())
	command.AddCommand(newPaymentsConnectorsDeprecatedGetConfigCommand())
	command.AddCommand(newPaymentsConnectorsDeprecatedUpdateConfigCommand())
	command.AddCommand(newPaymentsConnectorsUninstallCommand())
	return command
}

func newPaymentsConnectorsInstallCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:     "install <connector>",
		Aliases: []string{"i"},
		Short:   "Install a payment connector",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 || len(args) == 2 {
				return nil
			}
			return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments connectors install requires --confirm")
			}
			if len(args) == 2 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[1]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments connectors install <connector> --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments connectors install requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.InstallConnectorService{
				Handlers: paymentscmd.SDKInstallConnectorHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureInstallConnector,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.InstallConnectorInput{
				Connector: args[0],
				Config:    data,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentConnectorInstalled(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm connector installation")
	command.Flags().StringVar(&file, "file", "", "JSON connector config file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsConnectorsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List payment connectors",
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.ListConnectorsService{
				Handlers: paymentscmd.SDKListConnectorsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureListConnectors,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ListConnectorsInput{
				PageSize: pageSize,
				Cursor:   cursor,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentConnectors(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsConnectorsConfigCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Manage payment connector configuration",
	}
	command.AddCommand(newPaymentsConnectorsConfigShowCommand("show", nil, false))
	command.AddCommand(newPaymentsConnectorsConfigShowCommand("get", []string{"g"}, true))
	command.AddCommand(newPaymentsConnectorsConfigUpdateCommand("update", nil, false))
	return command
}

func newPaymentsConnectorsConfigShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var provider string
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <connector-id>",
		Aliases: aliases,
		Short:   "Show a payment connector configuration",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments connectors config get has been deprecated, use payments connectors config show")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetConnectorConfigService{
				Handlers: paymentscmd.SDKGetConnectorConfigHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetConnectorConfig,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetConnectorConfigInput{
				ConnectorID: args[0],
				Provider:    provider,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentConnectorConfig(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments connectors config show"
	}
	command.Flags().StringVar(&provider, "provider", "", "Connector provider, required only when pinned to payments API v1")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsConnectorsDeprecatedGetConfigCommand() *cobra.Command {
	var connectorID string
	var provider string
	var apiVersion string

	command := &cobra.Command{
		Use:        "get-config",
		Short:      "Show a payment connector configuration",
		Deprecated: "use payments connectors config show",
		Args:       cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "Command payments connectors get-config has been deprecated, use payments connectors config show <connector-id>")
			if connectorID == "" {
				return fmt.Errorf("payments connectors get-config requires --connector-id")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetConnectorConfigService{
				Handlers: paymentscmd.SDKGetConnectorConfigHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetConnectorConfig,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetConnectorConfigInput{
				ConnectorID: connectorID,
				Provider:    provider,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentConnectorConfig(cmd, output)
		},
	}
	command.Flags().StringVar(&connectorID, "connector-id", "", "Connector ID")
	command.Flags().StringVar(&provider, "provider", "", "Connector provider, required only when pinned to payments API v1")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsConnectorsConfigUpdateCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var confirm bool
	var file string
	var provider string
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <connector-id>",
		Aliases: aliases,
		Short:   "Update a payment connector configuration",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 || len(args) == 2 {
				return nil
			}
			return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments connectors update-config has been deprecated, use payments connectors config update <connector-id> --file <path>|-")
			}
			if !confirm {
				return fmt.Errorf("payments connectors config update requires --confirm")
			}
			if len(args) == 2 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[1]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments connectors config update <connector-id> --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments connectors config update requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.UpdateConnectorConfigService{
				Handlers: paymentscmd.SDKUpdateConnectorConfigHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureUpdateConnectorConfig,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.UpdateConnectorConfigInput{
				ConnectorID: args[0],
				Provider:    provider,
				Config:      data,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentConnectorConfigUpdated(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments connectors config update"
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm connector configuration update")
	command.Flags().StringVar(&file, "file", "", "JSON connector config file, or - for stdin")
	command.Flags().StringVar(&provider, "provider", "", "Connector provider, required for payments API v1 and used to infer a missing provider in v3 payloads")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsConnectorsDeprecatedUpdateConfigCommand() *cobra.Command {
	var connectorID string
	command := newPaymentsConnectorsConfigUpdateCommand("update-config <connector>", []string{"uc"}, true)
	command.Use = "update-config <connector>"
	command.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 || len(args) == 2 {
			return nil
		}
		return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
	}
	originalRunE := command.RunE
	command.RunE = func(cmd *cobra.Command, args []string) error {
		if connectorID == "" {
			return fmt.Errorf("payments connectors update-config requires --connector-id")
		}
		providerFlag := cmd.Flags().Lookup("provider")
		if providerFlag != nil && providerFlag.Value.String() == "" {
			if err := cmd.Flags().Set("provider", args[0]); err != nil {
				return err
			}
		}
		args[0] = connectorID
		return originalRunE(cmd, args)
	}
	command.Flags().StringVar(&connectorID, "connector-id", "", "Connector ID")
	return command
}

func newPaymentsConnectorsUninstallCommand() *cobra.Command {
	var confirm bool
	var provider string
	var apiVersion string

	command := &cobra.Command{
		Use:     "uninstall <connector-id>",
		Aliases: []string{"un", "u"},
		Short:   "Uninstall a payment connector",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments connectors uninstall requires --confirm")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.UninstallConnectorService{
				Handlers: paymentscmd.SDKUninstallConnectorHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureUninstallConnector,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.UninstallConnectorInput{
				ConnectorID: args[0],
				Provider:    provider,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentConnectorUninstalled(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm connector uninstall")
	command.Flags().StringVar(&provider, "provider", "", "Connector provider, required only when pinned to payments API v1")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentConnectorInstalled(cmd *cobra.Command, output paymentscmd.InstallConnectorOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if output.ConnectorID != "" {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Connector %s installed with ID: %s", output.Connector, output.ConnectorID)))
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Connector %s installed.", output.Connector)))
	return err
}

func renderPaymentConnectors(cmd *cobra.Command, output paymentscmd.ListConnectorsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Connectors) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No payment connectors found."))
		return err
	}
	rows := make([][]string, 0, len(output.Connectors))
	for _, connector := range output.Connectors {
		rows = append(rows, []string{connector.Provider, connector.Name, connector.ID})
	}
	if err := writeStyledRows(cmd, []string{"Provider", "Name", "ID"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderPaymentConnectorConfig(cmd *cobra.Command, output paymentscmd.GetConnectorConfigOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	rows := []styledKeyValue{{Label: "Connector ID", Value: output.ConnectorID}}
	if output.Provider != "" {
		rows = append(rows, styledKeyValue{Label: "Provider", Value: output.Provider})
	}
	if err := writeStyledColonKeyValues(cmd, rows...); err != nil {
		return err
	}
	formatted := output.Config
	var indented bytes.Buffer
	if err := json.Indent(&indented, output.Config, "", "  "); err == nil {
		formatted = indented.Bytes()
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", formatted)
	return err
}

func renderPaymentConnectorConfigUpdated(cmd *cobra.Command, output paymentscmd.UpdateConnectorConfigOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Connector %s config updated.", output.ConnectorID)))
	return err
}

func renderPaymentConnectorUninstalled(cmd *cobra.Command, output paymentscmd.UninstallConnectorOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		if err := writeStyledColonKeyValues(cmd, styledKeyValue{Label: "Task ID", Value: output.TaskID}); err != nil {
			return err
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Connector %s uninstall scheduled.", output.ConnectorID)))
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Connector %s uninstalled.", output.ConnectorID)))
	return err
}

func newPaymentsTransferInitiationCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	command := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Manage payment transfer initiations",
	}
	if deprecated {
		command.Deprecated = "use payments transfer-initiation"
		command.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Command payments transfer_initiation has been deprecated, use payments transfer-initiation")
		}
	}
	command.AddCommand(newPaymentsTransferInitiationListCommand())
	command.AddCommand(newPaymentsTransferInitiationCreateCommand())
	command.AddCommand(newPaymentsTransferInitiationShowCommand("show", []string{"sh", "s"}, false))
	command.AddCommand(newPaymentsTransferInitiationShowCommand("get", []string{"g"}, true))
	command.AddCommand(newPaymentsTransferInitiationActionCommand("approve", []string{"a"}, "Approve a payment transfer initiation", paymentscmd.FeatureApprovePaymentInitiation, paymentscmd.SDKApprovePaymentInitiationHandlers, "approved", true))
	command.AddCommand(newPaymentsTransferInitiationActionCommand("reject", []string{"rj"}, "Reject a payment transfer initiation", paymentscmd.FeatureRejectPaymentInitiation, paymentscmd.SDKRejectPaymentInitiationHandlers, "rejected", true))
	command.AddCommand(newPaymentsTransferInitiationActionCommand("retry", []string{"r"}, "Retry a payment transfer initiation", paymentscmd.FeatureRetryPaymentInitiation, paymentscmd.SDKRetryPaymentInitiationHandlers, "queued for retry", true))
	command.AddCommand(newPaymentsTransferInitiationActionCommand("delete", []string{"d"}, "Delete a payment transfer initiation", paymentscmd.FeatureDeletePaymentInitiation, paymentscmd.SDKDeletePaymentInitiationHandlers, "deleted", true))
	command.AddCommand(newPaymentsTransferInitiationReverseCommand())
	command.AddCommand(newPaymentsTransferInitiationUpdateStatusCommand("update-status", []string{"u"}, false))
	command.AddCommand(newPaymentsTransferInitiationUpdateStatusCommand("update_status", nil, true))
	return command
}

func newPaymentsTasksCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "tasks",
		Aliases: []string{"t"},
		Short:   "Manage payment tasks",
	}
	command.AddCommand(newPaymentsTasksShowCommand("show", []string{"sh", "s"}, false))
	command.AddCommand(newPaymentsTasksShowCommand("get", nil, true))
	return command
}

func newPaymentsPoolsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "pools",
		Short: "Manage payment pools",
	}
	command.AddCommand(newPaymentsPoolsCreateCommand())
	command.AddCommand(newPaymentsPoolsListCommand())
	command.AddCommand(newPaymentsPoolsShowCommand("show", nil, false))
	command.AddCommand(newPaymentsPoolsShowCommand("get", []string{"g"}, true))
	command.AddCommand(newPaymentsPoolsDeleteCommand())
	command.AddCommand(newPaymentsPoolsAddAccountCommand())
	command.AddCommand(newPaymentsPoolsRemoveAccountCommand())
	command.AddCommand(newPaymentsPoolsUpdateQueryCommand())
	command.AddCommand(newPaymentsPoolsBalancesCommand())
	command.AddCommand(newPaymentsPoolsLatestBalancesCommand())
	return command
}

func newPaymentsPaymentsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "payments",
		Aliases: []string{"p"},
		Short:   "Manage payments",
	}
	command.AddCommand(newPaymentsPaymentsCreateCommand())
	command.AddCommand(newPaymentsPaymentsListCommand())
	command.AddCommand(newPaymentsPaymentsShowCommand("show", nil, false))
	command.AddCommand(newPaymentsPaymentsShowCommand("get", []string{"g"}, true))
	command.AddCommand(newPaymentsPaymentsSetMetadataCommand())
	return command
}

func newPaymentsBankAccountsCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	command := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "Manage payment bank accounts",
	}
	if deprecated {
		command.Deprecated = "use payments bank-accounts"
		command.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Command payments bank_accounts has been deprecated, use payments bank-accounts")
		}
	}
	command.AddCommand(newPaymentsBankAccountsCreateCommand())
	command.AddCommand(newPaymentsBankAccountsListCommand())
	command.AddCommand(newPaymentsBankAccountsShowCommand("show", nil, false))
	command.AddCommand(newPaymentsBankAccountsShowCommand("get", []string{"g"}, true))
	command.AddCommand(newPaymentsBankAccountsForwardCommand())
	command.AddCommand(newPaymentsBankAccountsSetMetadataCommand("set-metadata", nil, false))
	command.AddCommand(newPaymentsBankAccountsSetMetadataCommand("update-metadata", []string{"update-meta"}, true))
	return command
}

func newPaymentsBankAccountsCreateCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr", "c"},
		Short:   "Create a payment bank account",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if len(args) == 1 {
				return nil
			}
			return fmt.Errorf("accepts 0 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments bank-accounts create requires --confirm")
			}
			if len(args) == 1 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[0]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments bank-accounts create --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments bank-accounts create requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
			if err != nil {
				return err
			}
			request, err := parseCreatePaymentBankAccountRequest(data)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.CreateBankAccountService{
				Handlers: paymentscmd.SDKCreateBankAccountHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureCreateBankAccount,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.CreateBankAccountInput{
				AccountNumber: request.AccountNumber,
				ConnectorID:   request.ConnectorID,
				Country:       request.Country,
				Iban:          request.Iban,
				Metadata:      request.Metadata,
				Name:          request.Name,
				SwiftBicCode:  request.SwiftBicCode,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentBankAccountCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm bank account creation")
	command.Flags().StringVar(&file, "file", "", "JSON bank account request file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

type createPaymentBankAccountRequestFile struct {
	AccountNumber string            `json:"accountNumber"`
	ConnectorID   string            `json:"connectorID"`
	Country       string            `json:"country"`
	Iban          string            `json:"iban"`
	Metadata      map[string]string `json:"metadata"`
	Name          string            `json:"name"`
	SwiftBicCode  string            `json:"swiftBicCode"`
}

func parseCreatePaymentBankAccountRequest(data []byte) (createPaymentBankAccountRequestFile, error) {
	var request createPaymentBankAccountRequestFile
	if err := json.Unmarshal(data, &request); err != nil {
		return createPaymentBankAccountRequestFile{}, err
	}
	return request, nil
}

func renderPaymentBankAccountCreated(cmd *cobra.Command, output paymentscmd.CreateBankAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Bank account created with ID: %s", output.BankAccountID)))
	return err
}

func newPaymentsBankAccountsForwardCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "forward <bank-account-id> <connector-id>",
		Aliases: []string{"fo", "f"},
		Short:   "Forward a payment bank account to a connector",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments bank-accounts forward requires --confirm")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.ForwardBankAccountService{
				Handlers: paymentscmd.SDKForwardBankAccountHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureForwardBankAccount,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ForwardBankAccountInput{
				BankAccountID: args[0],
				ConnectorID:   args[1],
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentBankAccountForwarded(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm bank account forwarding")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentBankAccountForwarded(cmd *cobra.Command, output paymentscmd.ForwardBankAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Bank account forwarding scheduled with task ID: %s", output.TaskID)))
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Bank account %s forwarded to connector %s.", output.BankAccountID, output.ConnectorID)))
	return err
}

func newPaymentsBankAccountsSetMetadataCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <bank-account-id> <key=value>...",
		Aliases: aliases,
		Short:   "Set payment bank account metadata",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments bank-accounts update-metadata has been deprecated, use payments bank-accounts set-metadata")
			}
			if !confirm {
				return fmt.Errorf("payments bank-accounts %s requires --confirm", use)
			}
			metadata, err := parseMetadataFlags(args[1:])
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.SetBankAccountMetadataService{
				Handlers: paymentscmd.SDKSetBankAccountMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureSetBankAccountMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.SetBankAccountMetadataInput{BankAccountID: args[0], Metadata: metadata})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentBankAccountMetadataSet(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments bank-accounts set-metadata"
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm bank account metadata update")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentBankAccountMetadataSet(cmd *cobra.Command, output paymentscmd.SetBankAccountMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Metadata set on bank account %s.", output.BankAccountID)))
	return err
}

func newPaymentsAccountsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "accounts",
		Short: "Manage payment accounts",
	}
	command.AddCommand(newPaymentsAccountsCreateCommand())
	command.AddCommand(newPaymentsAccountsListCommand())
	command.AddCommand(newPaymentsAccountsShowCommand("show", nil, false))
	command.AddCommand(newPaymentsAccountsShowCommand("get", []string{"g"}, true))
	command.AddCommand(newPaymentsAccountsBalancesCommand())
	return command
}

func newPaymentsAccountsCreateCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr", "c"},
		Short:   "Create a payment account",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if len(args) == 1 {
				return nil
			}
			return fmt.Errorf("accepts 0 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments accounts create requires --confirm")
			}
			if len(args) == 1 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[0]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments accounts create --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments accounts create requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
			if err != nil {
				return err
			}
			request, err := parseCreatePaymentAccountRequest(data)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.CreateAccountService{
				Handlers: paymentscmd.SDKCreateAccountHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureCreateAccount,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.CreateAccountInput{
				AccountName:  request.AccountName,
				ConnectorID:  request.ConnectorID,
				CreatedAt:    request.CreatedAt,
				DefaultAsset: request.DefaultAsset,
				Metadata:     request.Metadata,
				Reference:    request.Reference,
				Type:         request.Type,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentAccountCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm account creation")
	command.Flags().StringVar(&file, "file", "", "JSON account request file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

type createPaymentAccountRequestFile struct {
	AccountName  string            `json:"accountName"`
	ConnectorID  string            `json:"connectorID"`
	CreatedAt    time.Time         `json:"createdAt"`
	DefaultAsset string            `json:"defaultAsset"`
	Metadata     map[string]string `json:"metadata"`
	Reference    string            `json:"reference"`
	Type         string            `json:"type"`
}

func parseCreatePaymentAccountRequest(data []byte) (createPaymentAccountRequestFile, error) {
	var request createPaymentAccountRequestFile
	if err := json.Unmarshal(data, &request); err != nil {
		return createPaymentAccountRequestFile{}, err
	}
	request.Type = strings.ToUpper(request.Type)
	return request, nil
}

func renderPaymentAccountCreated(cmd *cobra.Command, output paymentscmd.CreateAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Account created with ID: %s", output.AccountID)))
	return err
}

func newPaymentsAccountsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List payment accounts",
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
			service := paymentscmd.ListAccountsService{
				Handlers: paymentscmd.SDKListAccountsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureListAccounts,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ListAccountsInput{
				PageSize: pageSize,
				Cursor:   cursor,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentAccounts(cmd, output)
		},
	}

	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")

	return command
}

func newPaymentsAccountsShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <account-id>",
		Aliases: aliases,
		Short:   "Show a payment account",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments accounts get has been deprecated, use payments accounts show")
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
			service := paymentscmd.GetAccountService{
				Handlers: paymentscmd.SDKGetAccountHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetAccount,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetAccountInput{AccountID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentAccount(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments accounts show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsAccountsBalancesCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var asset string
	var apiVersion string

	command := &cobra.Command{
		Use:   "balances <account-id>",
		Short: "List payment account balances",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			service := paymentscmd.ListAccountBalancesService{
				Handlers: paymentscmd.SDKListAccountBalancesHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetAccountBalances,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ListAccountBalancesInput{
				AccountID: args[0],
				PageSize:  pageSize,
				Cursor:    cursor,
				Asset:     asset,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentAccountBalances(cmd, output)
		},
	}

	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&asset, "asset", "", "Filter by asset")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")

	return command
}

func renderPaymentAccounts(cmd *cobra.Command, output paymentscmd.ListAccountsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Accounts) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No payment accounts found."))
		return err
	}
	rows := make([][]string, 0, len(output.Accounts))
	for _, account := range output.Accounts {
		rows = append(rows, []string{account.ID, account.Reference, account.CreatedAt.Format(time.RFC3339), account.Name, account.DefaultAsset, account.ConnectorID})
	}
	if err := writeStyledRows(cmd, []string{"ID", "Reference", "Created at", "Name", "Default asset", "Connector"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderPaymentAccountBalances(cmd *cobra.Command, output paymentscmd.ListAccountBalancesOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Balances) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No account balances found."))
		return err
	}
	rows := make([][]string, 0, len(output.Balances))
	for _, balance := range output.Balances {
		rows = append(rows, []string{balance.AccountID, balance.Asset, balance.Balance, balance.CreatedAt.Format(time.RFC3339), balance.LastUpdatedAt.Format(time.RFC3339)})
	}
	if err := writeStyledRows(cmd, []string{"Account", "Asset", "Balance", "Created at", "Updated at"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderPaymentAccount(cmd *cobra.Command, output paymentscmd.GetAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	account := output.Account
	return writeStyledKeyValues(cmd,
		styledKeyValue{Label: "ID", Value: account.ID},
		styledKeyValue{Label: "Reference", Value: account.Reference},
		styledKeyValue{Label: "Name", Value: account.Name},
		styledKeyValue{Label: "Created at", Value: account.CreatedAt.Format(time.RFC3339)},
		styledKeyValue{Label: "Connector ID", Value: account.ConnectorID},
		styledKeyValue{Label: "Default asset", Value: account.DefaultAsset},
		styledKeyValue{Label: "Type", Value: account.Type},
	)
}

func newPaymentsBankAccountsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List payment bank accounts",
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.ListBankAccountsService{
				Handlers: paymentscmd.SDKListBankAccountsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureListBankAccounts,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ListBankAccountsInput{PageSize: pageSize, Cursor: cursor})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentBankAccounts(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsBankAccountsShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <bank-account-id>",
		Aliases: aliases,
		Short:   "Show a payment bank account",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments bank-accounts get has been deprecated, use payments bank-accounts show")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetBankAccountService{
				Handlers: paymentscmd.SDKGetBankAccountHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetBankAccount,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetBankAccountInput{BankAccountID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentBankAccount(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments bank-accounts show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentBankAccounts(cmd *cobra.Command, output paymentscmd.ListBankAccountsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.BankAccounts) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No bank accounts found."))
		return err
	}
	rows := make([][]string, 0, len(output.BankAccounts))
	for _, account := range output.BankAccounts {
		rows = append(rows, []string{account.ID, account.Name, account.CreatedAt.Format(time.RFC3339), account.Country})
	}
	if err := writeStyledRows(cmd, []string{"ID", "Name", "Created at", "Country"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderPaymentBankAccount(cmd *cobra.Command, output paymentscmd.GetBankAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	account := output.BankAccount
	rows := []styledKeyValue{
		{Label: "ID", Value: account.ID},
		{Label: "Name", Value: account.Name},
		{Label: "Created at", Value: account.CreatedAt.Format(time.RFC3339)},
	}
	if account.Country != "" {
		rows = append(rows, styledKeyValue{Label: "Country", Value: account.Country})
	}
	if account.Iban != "" {
		rows = append(rows, styledKeyValue{Label: "IBAN", Value: account.Iban})
	}
	rows = append(rows, styledKeyValue{Label: "Swift BIC", Value: account.SwiftBicCode})
	return writeStyledKeyValues(cmd, rows...)
}

func newPaymentsPaymentsCreateCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr", "c"},
		Short:   "Create a payment",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if len(args) == 1 {
				return nil
			}
			return fmt.Errorf("accepts 0 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments payments create requires --confirm")
			}
			if len(args) == 1 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[0]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments payments create --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments payments create requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
			if err != nil {
				return err
			}
			request, err := parseCreatePaymentRequest(data)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.CreatePaymentService{
				Handlers: paymentscmd.SDKCreatePaymentHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureCreatePayment,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), request)
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm payment creation")
	command.Flags().StringVar(&file, "file", "", "JSON payment request file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func parseCreatePaymentRequest(data []byte) (paymentscmd.CreatePaymentInput, error) {
	var v1 shared.PaymentRequest
	if err := json.Unmarshal(data, &v1); err != nil {
		return paymentscmd.CreatePaymentInput{}, err
	}
	var v3 shared.V3CreatePaymentRequest
	if err := json.Unmarshal(data, &v3); err != nil {
		return paymentscmd.CreatePaymentInput{}, err
	}
	if v3.InitialAmount == nil {
		v3.InitialAmount = v3.Amount
	}
	return paymentscmd.CreatePaymentInput{V1: v1, V3: v3}, nil
}

func renderPaymentCreated(cmd *cobra.Command, output paymentscmd.CreatePaymentOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Payment created with ID: %s", output.PaymentID)))
	return err
}

func newPaymentsPaymentsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List payments",
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.ListPaymentsService{
				Handlers: paymentscmd.SDKListPaymentsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureListPayments,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ListPaymentsInput{PageSize: pageSize, Cursor: cursor})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPayments(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPaymentsShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <payment-id>",
		Aliases: aliases,
		Short:   "Show a payment",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments payments get has been deprecated, use payments payments show")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetPaymentService{
				Handlers: paymentscmd.SDKGetPaymentHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetPayment,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetPaymentInput{PaymentID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPayment(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments payments show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPaymentsSetMetadataCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "set-metadata <payment-id> <key=value>...",
		Aliases: []string{"sm", "set-meta"},
		Short:   "Set payment metadata",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments payments set-metadata requires --confirm")
			}
			metadata, err := parseMetadataFlags(args[1:])
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.SetPaymentMetadataService{
				Handlers: paymentscmd.SDKSetPaymentMetadataHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureSetPaymentMetadata,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.SetPaymentMetadataInput{PaymentID: args[0], Metadata: metadata})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentMetadataSet(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm payment metadata update")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentMetadataSet(cmd *cobra.Command, output paymentscmd.SetPaymentMetadataOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Metadata set on payment %s.", output.PaymentID)))
	return err
}

func renderPayments(cmd *cobra.Command, output paymentscmd.ListPaymentsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Payments) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No payments found."))
		return err
	}
	rows := make([][]string, 0, len(output.Payments))
	for _, payment := range output.Payments {
		rows = append(rows, []string{payment.ID, payment.Type, payment.Amount, payment.Asset, payment.Status, payment.CreatedAt.Format(time.RFC3339)})
	}
	if err := writeStyledRows(cmd, []string{"ID", "Type", "Amount", "Asset", "Status", "Created at"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderPayment(cmd *cobra.Command, output paymentscmd.GetPaymentOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	payment := output.Payment
	return writeStyledKeyValues(cmd,
		styledKeyValue{Label: "ID", Value: payment.ID},
		styledKeyValue{Label: "Reference", Value: payment.Reference},
		styledKeyValue{Label: "Amount", Value: payment.Amount},
		styledKeyValue{Label: "Asset", Value: payment.Asset},
		styledKeyValue{Label: "Status", Value: payment.Status},
		styledKeyValue{Label: "Created at", Value: payment.CreatedAt.Format(time.RFC3339)},
	)
}

func newPaymentsPoolsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var query string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List payment pools",
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.ListPoolsService{
				Handlers: paymentscmd.SDKListPoolsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureListPools,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ListPoolsInput{PageSize: pageSize, Cursor: cursor, Query: query})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPools(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&query, "query", "", "Filter pools with the API query syntax")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPoolsCreateCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr", "c"},
		Short:   "Create a payment pool",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || len(args) == 1 {
				return nil
			}
			return fmt.Errorf("accepts 0 or 1 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments pools create requires --confirm")
			}
			if len(args) == 1 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[0]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments pools create --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments pools create requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.CreatePoolService{
				Handlers: paymentscmd.SDKCreatePoolHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureCreatePool,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.CreatePoolInput{Payload: data})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPoolCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm pool creation")
	command.Flags().StringVar(&file, "file", "", "JSON pool request file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPoolsShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <pool-id>",
		Aliases: aliases,
		Short:   "Show a payment pool",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments pools get has been deprecated, use payments pools show")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetPoolService{
				Handlers: paymentscmd.SDKGetPoolHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetPool,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetPoolInput{PoolID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPool(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments pools show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPoolsDeleteCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "delete <pool-id>",
		Short: "Delete a payment pool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments pools delete requires --confirm")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.DeletePoolService{
				Handlers: paymentscmd.SDKDeletePoolHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureDeletePool,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.DeletePoolInput{PoolID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPoolDeleted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm pool deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentPoolCreated(cmd *cobra.Command, output paymentscmd.CreatePoolOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Pool created with ID: %s", output.PoolID)))
	return err
}

func renderPaymentPools(cmd *cobra.Command, output paymentscmd.ListPoolsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Pools) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No payment pools found."))
		return err
	}
	rows := make([][]string, 0, len(output.Pools))
	for _, pool := range output.Pools {
		rows = append(rows, []string{pool.ID, pool.Name, strings.Join(pool.Accounts, ",")})
	}
	if err := writeStyledRows(cmd, []string{"ID", "Name", "Accounts"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderPaymentPool(cmd *cobra.Command, output paymentscmd.GetPoolOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	pool := output.Pool
	rows := []styledKeyValue{
		{Label: "ID", Value: pool.ID},
		{Label: "Name", Value: pool.Name},
		{Label: "Accounts", Value: strings.Join(pool.Accounts, ",")},
	}
	if pool.Type != "" {
		rows = append(rows, styledKeyValue{Label: "Type", Value: pool.Type})
	}
	if !pool.CreatedAt.IsZero() {
		rows = append(rows, styledKeyValue{Label: "Created at", Value: pool.CreatedAt.Format(time.RFC3339)})
	}
	return writeStyledKeyValues(cmd, rows...)
}

func renderPaymentPoolDeleted(cmd *cobra.Command, output paymentscmd.DeletePoolOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Pool %s deleted.", output.PoolID)))
	return err
}

func newPaymentsPoolsAddAccountCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     "add-account <pool-id> <account-id>",
		Aliases: []string{"add", "a"},
		Short:   "Add an account to a payment pool",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.AddAccountToPoolService{
				Handlers: paymentscmd.SDKAddAccountToPoolHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureAddAccountToPool,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.PoolAccountInput{PoolID: args[0], AccountID: args[1]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPoolAccountAdded(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPoolsRemoveAccountCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "remove-account <pool-id> <account-id>",
		Aliases: []string{"remove", "rm"},
		Short:   "Remove an account from a payment pool",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments pools remove-account requires --confirm")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.RemoveAccountFromPoolService{
				Handlers: paymentscmd.SDKRemoveAccountFromPoolHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureRemoveAccountFromPool,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.PoolAccountInput{PoolID: args[0], AccountID: args[1]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPoolAccountRemoved(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm account removal")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentPoolAccountAdded(cmd *cobra.Command, output paymentscmd.PoolAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Account %s added to pool %s.", output.AccountID, output.PoolID)))
	return err
}

func renderPaymentPoolAccountRemoved(cmd *cobra.Command, output paymentscmd.PoolAccountOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Account %s removed from pool %s.", output.AccountID, output.PoolID)))
	return err
}

func newPaymentsPoolsUpdateQueryCommand() *cobra.Command {
	var file string
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     "update-query <pool-id> [file]",
		Aliases: []string{"uq"},
		Short:   "Update a payment pool query",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments pools update-query requires --confirm")
			}
			if len(args) == 2 {
				if file != "" {
					return fmt.Errorf("positional file and --file are mutually exclusive")
				}
				file = args[1]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments pools update-query <pool-id> --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments pools update-query requires --file <path>|-")
			}

			data, err := readPaymentCommandFile(cmd, file)
			if err != nil {
				return err
			}
			var request struct {
				Query map[string]any `json:"query"`
			}
			if err := json.Unmarshal(data, &request); err != nil {
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.UpdatePoolQueryService{
				Handlers: paymentscmd.SDKUpdatePoolQueryHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureUpdatePoolQuery,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.UpdatePoolQueryInput{PoolID: args[0], Query: request.Query})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPoolQueryUpdated(cmd, output)
		},
	}
	command.Flags().StringVar(&file, "file", "", "Read query payload from path or stdin with -")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm pool query update")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPoolsBalancesCommand() *cobra.Command {
	var at string
	var apiVersion string

	command := &cobra.Command{
		Use:   "balances <pool-id> [at]",
		Short: "List payment pool balances at a time",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				if at != "" {
					return fmt.Errorf("positional at and --at are mutually exclusive")
				}
				at = args[1]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional <at> has been deprecated, use payments pools balances <pool-id> --at <time>")
			}
			if at == "" {
				return fmt.Errorf("payments pools balances requires --at <time>")
			}
			parsedAt, err := time.Parse(time.RFC3339, at)
			if err != nil {
				return fmt.Errorf("parse --at as RFC3339: %w", err)
			}

			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetPoolBalancesService{
				Handlers: paymentscmd.SDKGetPoolBalancesHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetPoolBalances,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetPoolBalancesInput{PoolID: args[0], At: parsedAt})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPoolBalances(cmd, output)
		},
	}
	command.Flags().StringVar(&at, "at", "", "RFC3339 timestamp")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsPoolsLatestBalancesCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:   "latest-balances <pool-id>",
		Short: "List latest payment pool balances",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetPoolBalancesService{
				Handlers: paymentscmd.SDKGetPoolBalancesHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetPoolBalancesLatest,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetPoolBalancesInput{PoolID: args[0], Latest: true})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentPoolBalances(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func readPaymentCommandFile(cmd *cobra.Command, file string) ([]byte, error) {
	if file == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(file)
}

func renderPaymentPoolQueryUpdated(cmd *cobra.Command, output paymentscmd.UpdatePoolQueryOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Query updated for pool %s.", output.PoolID)))
	return err
}

func renderPaymentPoolBalances(cmd *cobra.Command, output paymentscmd.GetPoolBalancesOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Balances) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No pool balances found."))
		return err
	}
	rows := make([][]string, 0, len(output.Balances))
	for _, balance := range output.Balances {
		rows = append(rows, []string{balance.Asset, balance.Amount, strings.Join(balance.RelatedAccounts, ",")})
	}
	return writeStyledRows(cmd, []string{"Asset", "Amount", "Related accounts"}, rows)
}

func newPaymentsTasksShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <task-id>",
		Aliases: aliases,
		Short:   "Show a payment task",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments tasks get has been deprecated, use payments tasks show")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetTaskService{
				Handlers: paymentscmd.SDKGetTaskHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetTask,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetTaskInput{TaskID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentTask(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments tasks show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentTask(cmd *cobra.Command, output paymentscmd.GetTaskOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	task := output.Task
	rows := []styledKeyValue{{Label: "ID", Value: task.ID}}
	if task.ConnectorID != "" {
		rows = append(rows, styledKeyValue{Label: "Connector ID", Value: task.ConnectorID})
	}
	if task.CreatedObjectID != "" {
		rows = append(rows, styledKeyValue{Label: "Created object ID", Value: task.CreatedObjectID})
	}
	rows = append(rows, styledKeyValue{Label: "Status", Value: task.Status})
	if task.Error != "" {
		rows = append(rows, styledKeyValue{Label: "Error", Value: task.Error})
	}
	rows = append(rows,
		styledKeyValue{Label: "Created at", Value: task.CreatedAt.Format(time.RFC3339)},
		styledKeyValue{Label: "Updated at", Value: task.UpdatedAt.Format(time.RFC3339)},
	)
	return writeStyledKeyValues(cmd, rows...)
}

func newPaymentsTransferInitiationListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var query string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List payment transfer initiations",
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.ListTransferInitiationsService{
				Handlers: paymentscmd.SDKListTransferInitiationsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureListTransferInitiation,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ListTransferInitiationsInput{PageSize: pageSize, Cursor: cursor, Query: query})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentTransferInitiations(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&query, "query", "", "Filter transfer initiations with the API query syntax")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func newPaymentsTransferInitiationCreateCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr", "c"},
		Short:   "Create a payment transfer initiation",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if len(args) == 1 {
				return nil
			}
			return fmt.Errorf("accepts 0 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments transfer-initiation create requires --confirm")
			}
			if len(args) == 1 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[0]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments transfer-initiation create --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments transfer-initiation create requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
			if err != nil {
				return err
			}
			request, err := parseCreateTransferInitiationRequest(data)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.CreateTransferInitiationService{
				Handlers: paymentscmd.SDKCreateTransferInitiationHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureCreateTransferInitiation,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.CreateTransferInitiationInput{
				Amount:               request.Amount,
				Asset:                request.Asset,
				ConnectorID:          request.ConnectorID,
				Description:          request.Description,
				DestinationAccountID: request.DestinationAccountID,
				Metadata:             request.Metadata,
				Reference:            request.Reference,
				ScheduledAt:          request.ScheduledAt,
				SourceAccountID:      request.SourceAccountID,
				Type:                 request.Type,
				Validated:            request.Validated,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentTransferInitiationCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm transfer initiation creation")
	command.Flags().StringVar(&file, "file", "", "JSON transfer initiation request file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

type createTransferInitiationRequestFile struct {
	Amount               *big.Int          `json:"amount"`
	Asset                string            `json:"asset"`
	ConnectorID          string            `json:"connectorID"`
	Description          string            `json:"description"`
	DestinationAccountID string            `json:"destinationAccountID"`
	Metadata             map[string]string `json:"metadata"`
	Reference            string            `json:"reference"`
	ScheduledAt          time.Time         `json:"scheduledAt"`
	SourceAccountID      string            `json:"sourceAccountID"`
	Type                 string            `json:"type"`
	Validated            bool              `json:"validated"`
}

func parseCreateTransferInitiationRequest(data []byte) (createTransferInitiationRequestFile, error) {
	var raw struct {
		Amount               json.RawMessage   `json:"amount"`
		Asset                string            `json:"asset"`
		ConnectorID          string            `json:"connectorID"`
		Description          string            `json:"description"`
		DestinationAccountID string            `json:"destinationAccountID"`
		Metadata             map[string]string `json:"metadata"`
		Reference            string            `json:"reference"`
		ScheduledAt          time.Time         `json:"scheduledAt"`
		SourceAccountID      string            `json:"sourceAccountID"`
		Type                 string            `json:"type"`
		Validated            bool              `json:"validated"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return createTransferInitiationRequestFile{}, err
	}
	if len(raw.Amount) == 0 {
		return createTransferInitiationRequestFile{}, fmt.Errorf("transfer initiation amount is required")
	}
	amount, err := parseBigIntJSON(raw.Amount)
	if err != nil {
		return createTransferInitiationRequestFile{}, fmt.Errorf("invalid transfer initiation amount: %w", err)
	}
	return createTransferInitiationRequestFile{
		Amount:               amount,
		Asset:                raw.Asset,
		ConnectorID:          raw.ConnectorID,
		Description:          raw.Description,
		DestinationAccountID: raw.DestinationAccountID,
		Metadata:             raw.Metadata,
		Reference:            raw.Reference,
		ScheduledAt:          raw.ScheduledAt,
		SourceAccountID:      raw.SourceAccountID,
		Type:                 strings.ToUpper(raw.Type),
		Validated:            raw.Validated,
	}, nil
}

func renderPaymentTransferInitiationCreated(cmd *cobra.Command, output paymentscmd.CreateTransferInitiationOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		if err := writeStyledColonKeyValues(cmd, styledKeyValue{Label: "Task ID", Value: output.TaskID}); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Transfer initiation created with ID: %s", output.TransferInitiationID)))
	return err
}

func newPaymentsTransferInitiationShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <transfer-initiation-id>",
		Aliases: aliases,
		Short:   "Show a payment transfer initiation",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments transfer-initiation get has been deprecated, use payments transfer-initiation show")
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.GetTransferInitiationService{
				Handlers: paymentscmd.SDKGetTransferInitiationHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureGetTransferInitiation,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.GetTransferInitiationInput{TransferInitiationID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentTransferInitiation(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments transfer-initiation show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentTransferInitiations(cmd *cobra.Command, output paymentscmd.ListTransferInitiationsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.TransferInitiations) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No transfer initiations found."))
		return err
	}
	rows := make([][]string, 0, len(output.TransferInitiations))
	for _, transfer := range output.TransferInitiations {
		rows = append(rows, []string{transfer.ID, transfer.Type, transfer.Amount, transfer.Asset, transfer.Status, transfer.CreatedAt.Format(time.RFC3339)})
	}
	if err := writeStyledRows(cmd, []string{"ID", "Type", "Amount", "Asset", "Status", "Created at"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderPaymentTransferInitiation(cmd *cobra.Command, output paymentscmd.GetTransferInitiationOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	transfer := output.TransferInitiation
	return writeStyledKeyValues(cmd,
		styledKeyValue{Label: "ID", Value: transfer.ID},
		styledKeyValue{Label: "Reference", Value: transfer.Reference},
		styledKeyValue{Label: "Amount", Value: transfer.Amount},
		styledKeyValue{Label: "Asset", Value: transfer.Asset},
		styledKeyValue{Label: "Status", Value: transfer.Status},
		styledKeyValue{Label: "Connector ID", Value: transfer.ConnectorID},
		styledKeyValue{Label: "Created at", Value: transfer.CreatedAt.Format(time.RFC3339)},
	)
}

func newPaymentsTransferInitiationActionCommand(
	use string,
	aliases []string,
	short string,
	feature capabilities.Feature,
	handlers func(*formance.Formance) []paymentscmd.TransferInitiationActionHandler,
	done string,
	requiresConfirm bool,
) *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <transfer-initiation-id>",
		Aliases: aliases,
		Short:   short,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if requiresConfirm && !confirm {
				return fmt.Errorf("payments transfer-initiation %s requires --confirm", use)
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.TransferInitiationActionService{
				Handlers: handlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         feature,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.TransferInitiationActionInput{TransferInitiationID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentTransferInitiationAction(cmd, output, done)
		},
	}
	if requiresConfirm {
		command.Flags().BoolVar(&confirm, "confirm", false, "Confirm transfer initiation action")
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentTransferInitiationAction(cmd *cobra.Command, output paymentscmd.TransferInitiationActionOutput, done string) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		if err := writeStyledColonKeyValues(cmd, styledKeyValue{Label: "Task ID", Value: output.TaskID}); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Transfer initiation %s %s.", output.TransferInitiationID, done)))
	return err
}

func newPaymentsTransferInitiationUpdateStatusCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <transfer-initiation-id> <status>",
		Aliases: aliases,
		Short:   "Update a payment transfer initiation status",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command payments transfer-initiation update_status has been deprecated, use payments transfer-initiation update-status")
			}
			if !confirm {
				return fmt.Errorf("payments transfer-initiation %s requires --confirm", use)
			}
			status := strings.ToUpper(args[1])
			if status != "REJECTED" && status != "VALIDATED" {
				return fmt.Errorf("unsupported transfer initiation status %q: expected REJECTED or VALIDATED", args[1])
			}
			rt, err := stackRuntimeFromCommand(cmd)
			if err != nil {
				return err
			}
			httpClient, err := rt.HTTPClient(cmd.Context())
			if err != nil {
				return err
			}
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.UpdateTransferInitiationStatusService{
				Handlers: paymentscmd.SDKUpdateTransferInitiationStatusHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureUpdateTransferInitiationStatus,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.UpdateTransferInitiationStatusInput{TransferInitiationID: args[0], Status: status})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentTransferInitiationStatusUpdated(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use payments transfer-initiation update-status"
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm transfer initiation status update")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

func renderPaymentTransferInitiationStatusUpdated(cmd *cobra.Command, output paymentscmd.UpdateTransferInitiationStatusOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Transfer initiation %s status updated to %s.", output.TransferInitiationID, output.Status)))
	return err
}

func newPaymentsTransferInitiationReverseCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:     "reverse <transfer-initiation-id>",
		Aliases: []string{"re"},
		Short:   "Reverse a payment transfer initiation",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return nil
			}
			if len(args) == 2 {
				return nil
			}
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("payments transfer-initiation reverse requires --confirm")
			}
			if len(args) == 2 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[1]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use payments transfer-initiation reverse <transfer-initiation-id> --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("payments transfer-initiation reverse requires --file <path>|-")
			}
			data, err := readPaymentCommandFile(cmd, file)
			if err != nil {
				return err
			}
			request, err := parseReverseTransferInitiationRequest(data)
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := paymentscmd.ReverseTransferInitiationService{
				Handlers: paymentscmd.SDKReverseTransferInitiationHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         paymentscmd.ProductPayments,
						Feature:         paymentscmd.FeatureReversePaymentInitiation,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), paymentscmd.ReverseTransferInitiationInput{
				TransferInitiationID: args[0],
				Amount:               request.Amount,
				Asset:                request.Asset,
				Description:          request.Description,
				Metadata:             request.Metadata,
				Reference:            request.Reference,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderPaymentTransferInitiationReversed(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm transfer initiation reversal")
	command.Flags().StringVar(&file, "file", "", "JSON reverse request file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin payments API version")
	return command
}

type reverseTransferInitiationRequestFile struct {
	Amount      *big.Int          `json:"amount"`
	Asset       string            `json:"asset"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Reference   string            `json:"reference"`
}

func parseReverseTransferInitiationRequest(data []byte) (reverseTransferInitiationRequestFile, error) {
	var raw struct {
		Amount      json.RawMessage   `json:"amount"`
		Asset       string            `json:"asset"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
		Reference   string            `json:"reference"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return reverseTransferInitiationRequestFile{}, err
	}
	if len(raw.Amount) == 0 {
		return reverseTransferInitiationRequestFile{}, fmt.Errorf("reverse transfer initiation amount is required")
	}
	amount, err := parseBigIntJSON(raw.Amount)
	if err != nil {
		return reverseTransferInitiationRequestFile{}, fmt.Errorf("invalid reverse transfer initiation amount: %w", err)
	}
	return reverseTransferInitiationRequestFile{
		Amount:      amount,
		Asset:       raw.Asset,
		Description: raw.Description,
		Metadata:    raw.Metadata,
		Reference:   raw.Reference,
	}, nil
}

func parseBigIntJSON(data []byte) (*big.Int, error) {
	value := strings.TrimSpace(string(data))
	value = strings.Trim(value, `"`)
	if value == "" {
		return nil, fmt.Errorf("empty integer")
	}
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("expected integer")
	}
	return amount, nil
}

func renderPaymentTransferInitiationReversed(cmd *cobra.Command, output paymentscmd.ReverseTransferInitiationOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		if err := writeStyledColonKeyValues(cmd, styledKeyValue{Label: "Task ID", Value: output.TaskID}); err != nil {
			return err
		}
	}
	if output.ReversalID != "" {
		if err := writeStyledColonKeyValues(cmd, styledKeyValue{Label: "Reversal ID", Value: output.ReversalID}); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Transfer initiation %s reversed.", output.TransferInitiationID)))
	return err
}
