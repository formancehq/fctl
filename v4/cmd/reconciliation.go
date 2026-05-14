package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	reconciliationcmd "github.com/formancehq/fctl/v4/internal/commands/reconciliation"
)

func newReconciliationCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "reconciliation",
		Short: "Manage reconciliation policies and runs",
	}
	command.AddCommand(newReconciliationListCommand())
	command.AddCommand(newReconciliationShowCommand("show", []string{"sh"}, false))
	command.AddCommand(newReconciliationShowCommand("get", nil, true))
	command.AddCommand(newReconciliationPoliciesCommand())
	return command
}

func newReconciliationPoliciesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "policies",
		Short: "Manage reconciliation policies",
	}
	command.AddCommand(newReconciliationPoliciesListCommand())
	command.AddCommand(newReconciliationPoliciesCreateCommand())
	command.AddCommand(newReconciliationPoliciesShowCommand("show", []string{"sh"}, false))
	command.AddCommand(newReconciliationPoliciesShowCommand("get", nil, true))
	command.AddCommand(newReconciliationPoliciesDeleteCommand())
	command.AddCommand(newReconciliationPoliciesReconcileCommand())
	return command
}

func newReconciliationListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var policyID string
	var status string
	var query []string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List reconciliations",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			queryMap, err := parseReconciliationQueryFlags(query, map[string]string{
				"policyID": policyID,
				"status":   status,
			})
			if err != nil {
				return err
			}
			service, err := newListReconciliationsService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), reconciliationcmd.ListReconciliationsInput{
				PageSize: pageSize,
				Cursor:   cursor,
				Query:    queryMap,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderReconciliations(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&policyID, "policy-id", "", "Filter by policy ID")
	command.Flags().StringVar(&status, "status", "", "Filter by status")
	command.Flags().StringArrayVar(&query, "query", nil, "Query filter as key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin reconciliation API version")
	return command
}

func newReconciliationShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <reconciliation-id>",
		Aliases: aliases,
		Short:   "Show a reconciliation",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command reconciliation get has been deprecated, use reconciliation show")
			}
			service, err := newGetReconciliationService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), reconciliationcmd.GetReconciliationInput{ReconciliationID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderReconciliation(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use reconciliation show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin reconciliation API version")
	return command
}

func newReconciliationPoliciesListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var name string
	var ledger string
	var paymentsPoolID string
	var query []string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List reconciliation policies",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			queryMap, err := parseReconciliationQueryFlags(query, map[string]string{
				"name":           name,
				"ledgerName":     ledger,
				"paymentsPoolID": paymentsPoolID,
			})
			if err != nil {
				return err
			}
			service, err := newListPoliciesService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), reconciliationcmd.ListPoliciesInput{
				PageSize: pageSize,
				Cursor:   cursor,
				Query:    queryMap,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderReconciliationPolicies(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&name, "name", "", "Filter by policy name")
	command.Flags().StringVar(&ledger, "ledger", "", "Filter by ledger name")
	command.Flags().StringVar(&paymentsPoolID, "payments-pool-id", "", "Filter by payments pool ID")
	command.Flags().StringArrayVar(&query, "query", nil, "Query filter as key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin reconciliation API version")
	return command
}

func newReconciliationPoliciesCreateCommand() *cobra.Command {
	var confirm bool
	var file string
	var apiVersion string

	command := &cobra.Command{
		Use:   "create",
		Short: "Create a reconciliation policy",
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
				return fmt.Errorf("reconciliation policies create requires --confirm")
			}
			if len(args) == 1 {
				if file != "" {
					return fmt.Errorf("use either --file or positional file, not both")
				}
				file = args[0]
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional file has been deprecated, use reconciliation policies create --file <path>|-")
			}
			if file == "" {
				return fmt.Errorf("reconciliation policies create requires --file <path>|-")
			}
			payload, err := readReconciliationCommandFile(cmd, file)
			if err != nil {
				return err
			}
			service, err := newCreatePolicyService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), reconciliationcmd.CreatePolicyInput{Payload: payload})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderReconciliationPolicyCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm policy creation")
	command.Flags().StringVar(&file, "file", "", "JSON policy request file, or - for stdin")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin reconciliation API version")
	return command
}

func newReconciliationPoliciesShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <policy-id>",
		Aliases: aliases,
		Short:   "Show a reconciliation policy",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command reconciliation policies get has been deprecated, use reconciliation policies show")
			}
			service, err := newGetPolicyService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), reconciliationcmd.GetPolicyInput{PolicyID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderReconciliationPolicy(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use reconciliation policies show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin reconciliation API version")
	return command
}

func newReconciliationPoliciesDeleteCommand() *cobra.Command {
	var confirm bool
	var apiVersion string

	command := &cobra.Command{
		Use:   "delete <policy-id>",
		Short: "Delete a reconciliation policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("reconciliation policies delete requires --confirm")
			}
			service, err := newDeletePolicyService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), reconciliationcmd.DeletePolicyInput{PolicyID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderReconciliationPolicyDeleted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm policy deletion")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin reconciliation API version")
	return command
}

func newReconciliationPoliciesReconcileCommand() *cobra.Command {
	var confirm bool
	var ledgerAt string
	var paymentsAt string
	var apiVersion string

	command := &cobra.Command{
		Use:   "reconcile <policy-id> [ledger-at payments-at]",
		Short: "Run reconciliation for a policy",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 || len(args) == 3 {
				return nil
			}
			return fmt.Errorf("accepts either <policy-id> or deprecated <policy-id> <ledger-at> <payments-at>, received %d", len(args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("reconciliation policies reconcile requires --confirm")
			}
			if len(args) == 3 {
				if ledgerAt != "" || paymentsAt != "" {
					return fmt.Errorf("deprecated positional reconciliation timestamps cannot be combined with --ledger-at or --payments-at")
				}
				fmt.Fprintln(cmd.ErrOrStderr(), "Positional reconciliation timestamps have been deprecated, use --ledger-at and --payments-at")
				ledgerAt = args[1]
				paymentsAt = args[2]
			}
			parsedLedgerAt, err := parseOptionalRFC3339(ledgerAt, "ledger-at")
			if err != nil {
				return err
			}
			if parsedLedgerAt == nil {
				return fmt.Errorf("reconciliation policies reconcile requires --ledger-at")
			}
			parsedPaymentsAt, err := parseOptionalRFC3339(paymentsAt, "payments-at")
			if err != nil {
				return err
			}
			if parsedPaymentsAt == nil {
				return fmt.Errorf("reconciliation policies reconcile requires --payments-at")
			}
			service, err := newReconcileService(cmd, apiVersion)
			if err != nil {
				return err
			}
			output, err := service.Run(cmd.Context(), reconciliationcmd.ReconcileInput{
				PolicyID:             args[0],
				ReconciledAtLedger:   *parsedLedgerAt,
				ReconciledAtPayments: *parsedPaymentsAt,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderReconciliationStarted(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm reconciliation run")
	command.Flags().StringVar(&ledgerAt, "ledger-at", "", "Ledger reconciliation timestamp (RFC3339)")
	command.Flags().StringVar(&paymentsAt, "payments-at", "", "Payments reconciliation timestamp (RFC3339)")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin reconciliation API version")
	return command
}

func newCreatePolicyService(cmd *cobra.Command, apiVersion string) (reconciliationcmd.CreatePolicyService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return reconciliationcmd.CreatePolicyService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return reconciliationcmd.CreatePolicyService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return reconciliationcmd.CreatePolicyService{
		Handlers: reconciliationcmd.SDKCreatePolicyHandlers(sdk),
		Resolve:  reconciliationVersionResolver(rt, reconciliationcmd.FeatureCreatePolicy, apiVersion),
	}, nil
}

func newListPoliciesService(cmd *cobra.Command, apiVersion string) (reconciliationcmd.ListPoliciesService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return reconciliationcmd.ListPoliciesService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return reconciliationcmd.ListPoliciesService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return reconciliationcmd.ListPoliciesService{
		Handlers: reconciliationcmd.SDKListPoliciesHandlers(sdk),
		Resolve:  reconciliationVersionResolver(rt, reconciliationcmd.FeatureListPolicies, apiVersion),
	}, nil
}

func newGetPolicyService(cmd *cobra.Command, apiVersion string) (reconciliationcmd.GetPolicyService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return reconciliationcmd.GetPolicyService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return reconciliationcmd.GetPolicyService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return reconciliationcmd.GetPolicyService{
		Handlers: reconciliationcmd.SDKGetPolicyHandlers(sdk),
		Resolve:  reconciliationVersionResolver(rt, reconciliationcmd.FeatureGetPolicy, apiVersion),
	}, nil
}

func newDeletePolicyService(cmd *cobra.Command, apiVersion string) (reconciliationcmd.DeletePolicyService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return reconciliationcmd.DeletePolicyService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return reconciliationcmd.DeletePolicyService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return reconciliationcmd.DeletePolicyService{
		Handlers: reconciliationcmd.SDKDeletePolicyHandlers(sdk),
		Resolve:  reconciliationVersionResolver(rt, reconciliationcmd.FeatureDeletePolicy, apiVersion),
	}, nil
}

func newListReconciliationsService(cmd *cobra.Command, apiVersion string) (reconciliationcmd.ListReconciliationsService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return reconciliationcmd.ListReconciliationsService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return reconciliationcmd.ListReconciliationsService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return reconciliationcmd.ListReconciliationsService{
		Handlers: reconciliationcmd.SDKListReconciliationsHandlers(sdk),
		Resolve:  reconciliationVersionResolver(rt, reconciliationcmd.FeatureListReconciliations, apiVersion),
	}, nil
}

func newGetReconciliationService(cmd *cobra.Command, apiVersion string) (reconciliationcmd.GetReconciliationService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return reconciliationcmd.GetReconciliationService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return reconciliationcmd.GetReconciliationService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return reconciliationcmd.GetReconciliationService{
		Handlers: reconciliationcmd.SDKGetReconciliationHandlers(sdk),
		Resolve:  reconciliationVersionResolver(rt, reconciliationcmd.FeatureGetReconciliation, apiVersion),
	}, nil
}

func newReconcileService(cmd *cobra.Command, apiVersion string) (reconciliationcmd.ReconcileService, error) {
	rt, err := stackRuntimeFromCommand(cmd)
	if err != nil {
		return reconciliationcmd.ReconcileService{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return reconciliationcmd.ReconcileService{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	return reconciliationcmd.ReconcileService{
		Handlers: reconciliationcmd.SDKReconcileHandlers(sdk),
		Resolve:  reconciliationVersionResolver(rt, reconciliationcmd.FeatureReconcile, apiVersion),
	}, nil
}

func reconciliationVersionResolver(rt interface {
	ResolveAPIVersion(context.Context, capabilities.VersionResolutionRequest) (capabilities.APIVersion, error)
}, feature capabilities.Feature, apiVersion string) func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error) {
	return func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
		request := capabilities.VersionResolutionRequest{
			Product:         reconciliationcmd.ProductReconciliation,
			Feature:         feature,
			HandlerVersions: handlerVersions,
		}
		if apiVersion != "" {
			request.Policy = capabilities.VersionPolicyPinned
			request.PinnedVersion = capabilities.APIVersion(apiVersion)
		}
		return rt.ResolveAPIVersion(ctx, request)
	}
}

func readReconciliationCommandFile(cmd *cobra.Command, file string) ([]byte, error) {
	if file == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(file)
}

func parseReconciliationQueryFlags(values []string, filters map[string]string) (map[string]any, error) {
	query, err := parseAnyMapFlags(values)
	if err != nil {
		return nil, err
	}
	if len(query) == 0 {
		query = map[string]any{}
	}
	for key, value := range filters {
		if value != "" {
			query[key] = value
		}
	}
	if len(query) == 0 {
		return nil, nil
	}
	return query, nil
}

func renderReconciliationPolicyCreated(cmd *cobra.Command, output reconciliationcmd.CreatePolicyOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Policy created with ID: %s", output.Policy.ID)))
	return err
}

func renderReconciliationPolicies(cmd *cobra.Command, output reconciliationcmd.ListPoliciesOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Policies) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No reconciliation policies found."))
		return err
	}
	rows := make([][]string, 0, len(output.Policies))
	for _, policy := range output.Policies {
		rows = append(rows, []string{policy.ID, policy.Name, policy.LedgerName, policy.PaymentsPoolID})
	}
	if err := writeStyledRows(cmd, []string{"ID", "Name", "Ledger", "Payments pool ID"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderReconciliationPolicy(cmd *cobra.Command, output reconciliationcmd.GetPolicyOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	policy := output.Policy
	rows := []styledKeyValue{
		{Label: "ID", Value: policy.ID},
		{Label: "Name", Value: policy.Name},
		{Label: "Ledger", Value: policy.LedgerName},
		{Label: "Payments pool ID", Value: policy.PaymentsPoolID},
	}
	if !policy.CreatedAt.IsZero() {
		rows = append(rows, styledKeyValue{Label: "Created at", Value: policy.CreatedAt.Format(time.RFC3339)})
	}
	return writeStyledKeyValues(cmd, rows...)
}

func renderReconciliationPolicyDeleted(cmd *cobra.Command, output reconciliationcmd.DeletePolicyOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Policy %s deleted.", output.PolicyID)))
	return err
}

func renderReconciliationStarted(cmd *cobra.Command, output reconciliationcmd.ReconcileOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), styledSuccessLine(cmd, fmt.Sprintf("Reconciliation started with ID: %s", output.Reconciliation.ID)))
	return err
}

func renderReconciliations(cmd *cobra.Command, output reconciliationcmd.ListReconciliationsOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	if len(output.Reconciliations) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), styledEmptyLine(cmd, "No reconciliations found."))
		return err
	}
	rows := make([][]string, 0, len(output.Reconciliations))
	for _, reconciliation := range output.Reconciliations {
		rows = append(rows, []string{reconciliation.ID, reconciliation.PolicyID, reconciliation.Status, reconciliation.CreatedAt.Format(time.RFC3339)})
	}
	if err := writeStyledRows(cmd, []string{"ID", "Policy ID", "Status", "Created at"}, rows); err != nil {
		return err
	}
	if output.HasMore && output.Next != nil {
		return writeStyledNext(cmd, *output.Next)
	}
	return nil
}

func renderReconciliation(cmd *cobra.Command, output reconciliationcmd.GetReconciliationOutput) error {
	if err := writeStyledAPIVersion(cmd, output.APIVersion); err != nil {
		return err
	}
	reconciliation := output.Reconciliation
	rows := []styledKeyValue{
		{Label: "ID", Value: reconciliation.ID},
		{Label: "Policy ID", Value: reconciliation.PolicyID},
		{Label: "Status", Value: reconciliation.Status},
	}
	if reconciliation.Error != "" {
		rows = append(rows, styledKeyValue{Label: "Error", Value: reconciliation.Error})
	}
	if !reconciliation.ReconciledAtLedger.IsZero() {
		rows = append(rows, styledKeyValue{Label: "Reconciled at ledger", Value: reconciliation.ReconciledAtLedger.Format(time.RFC3339)})
	}
	if !reconciliation.ReconciledAtPayments.IsZero() {
		rows = append(rows, styledKeyValue{Label: "Reconciled at payments", Value: reconciliation.ReconciledAtPayments.Format(time.RFC3339)})
	}
	if !reconciliation.CreatedAt.IsZero() {
		rows = append(rows, styledKeyValue{Label: "Created at", Value: reconciliation.CreatedAt.Format(time.RFC3339)})
	}
	return writeStyledKeyValues(cmd, rows...)
}
