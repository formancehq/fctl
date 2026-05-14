package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
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
	command.AddCommand(newPaymentsPaymentsCommand())
	command.AddCommand(newPaymentsPoolsCommand())
	command.AddCommand(newPaymentsTasksCommand())
	command.AddCommand(newPaymentsTransferInitiationCommand("transfer-initiation", []string{"ti"}, false))
	command.AddCommand(newPaymentsTransferInitiationCommand("transfer_initiation", nil, true))
	return command
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
	command.AddCommand(newPaymentsPaymentsListCommand())
	command.AddCommand(newPaymentsPaymentsShowCommand("show", nil, false))
	command.AddCommand(newPaymentsPaymentsShowCommand("get", []string{"g"}, true))
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
	command.AddCommand(newPaymentsBankAccountsListCommand())
	command.AddCommand(newPaymentsBankAccountsShowCommand("show", nil, false))
	command.AddCommand(newPaymentsBankAccountsShowCommand("get", []string{"g"}, true))
	return command
}

func newPaymentsAccountsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "accounts",
		Short: "Manage payment accounts",
	}
	command.AddCommand(newPaymentsAccountsListCommand())
	command.AddCommand(newPaymentsAccountsShowCommand("show", nil, false))
	command.AddCommand(newPaymentsAccountsShowCommand("get", []string{"g"}, true))
	command.AddCommand(newPaymentsAccountsBalancesCommand())
	return command
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Accounts) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No payment accounts found.")
		return err
	}
	for _, account := range output.Accounts {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			account.ID,
			account.Reference,
			account.CreatedAt.Format(time.RFC3339),
			account.Name,
			account.DefaultAsset,
			account.ConnectorID,
		); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderPaymentAccountBalances(cmd *cobra.Command, output paymentscmd.ListAccountBalancesOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Balances) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No account balances found.")
		return err
	}
	for _, balance := range output.Balances {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s\t%s\t%s\t%s\t%s\n",
			balance.AccountID,
			balance.Asset,
			balance.Balance,
			balance.CreatedAt.Format(time.RFC3339),
			balance.LastUpdatedAt.Format(time.RFC3339),
		); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderPaymentAccount(cmd *cobra.Command, output paymentscmd.GetAccountOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	account := output.Account
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", account.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Reference\t%s\n", account.Reference); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", account.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", account.CreatedAt.Format(time.RFC3339)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Connector ID\t%s\n", account.ConnectorID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Default asset\t%s\n", account.DefaultAsset); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Type\t%s\n", account.Type)
	return err
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
			rt, err := runtimeFromCommand(cmd)
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.BankAccounts) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No bank accounts found.")
		return err
	}
	for _, account := range output.BankAccounts {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", account.ID, account.Name, account.CreatedAt.Format(time.RFC3339), account.Country); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderPaymentBankAccount(cmd *cobra.Command, output paymentscmd.GetBankAccountOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	account := output.BankAccount
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", account.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", account.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", account.CreatedAt.Format(time.RFC3339)); err != nil {
		return err
	}
	if account.Country != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Country\t%s\n", account.Country); err != nil {
			return err
		}
	}
	if account.Iban != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "IBAN\t%s\n", account.Iban); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Swift BIC\t%s\n", account.SwiftBicCode)
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
			rt, err := runtimeFromCommand(cmd)
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
			rt, err := runtimeFromCommand(cmd)
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

func renderPayments(cmd *cobra.Command, output paymentscmd.ListPaymentsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Payments) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No payments found.")
		return err
	}
	for _, payment := range output.Payments {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\t%s\n", payment.ID, payment.Type, payment.Amount, payment.Asset, payment.Status, payment.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderPayment(cmd *cobra.Command, output paymentscmd.GetPaymentOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	payment := output.Payment
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", payment.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Reference\t%s\n", payment.Reference); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Amount\t%s\n", payment.Amount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Asset\t%s\n", payment.Asset); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Status\t%s\n", payment.Status); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", payment.CreatedAt.Format(time.RFC3339))
	return err
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
			rt, err := runtimeFromCommand(cmd)
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
			rt, err := runtimeFromCommand(cmd)
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
			rt, err := runtimeFromCommand(cmd)
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

func renderPaymentPools(cmd *cobra.Command, output paymentscmd.ListPoolsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Pools) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No payment pools found.")
		return err
	}
	for _, pool := range output.Pools {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", pool.ID, pool.Name, strings.Join(pool.Accounts, ",")); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderPaymentPool(cmd *cobra.Command, output paymentscmd.GetPoolOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	pool := output.Pool
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", pool.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", pool.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Accounts\t%s\n", strings.Join(pool.Accounts, ",")); err != nil {
		return err
	}
	if pool.Type != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Type\t%s\n", pool.Type); err != nil {
			return err
		}
	}
	if !pool.CreatedAt.IsZero() {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", pool.CreatedAt.Format(time.RFC3339))
		return err
	}
	return nil
}

func renderPaymentPoolDeleted(cmd *cobra.Command, output paymentscmd.DeletePoolOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Pool %s deleted.\n", output.PoolID)
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
			rt, err := runtimeFromCommand(cmd)
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Account %s added to pool %s.\n", output.AccountID, output.PoolID)
	return err
}

func renderPaymentPoolAccountRemoved(cmd *cobra.Command, output paymentscmd.PoolAccountOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Account %s removed from pool %s.\n", output.AccountID, output.PoolID)
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

			rt, err := runtimeFromCommand(cmd)
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

			rt, err := runtimeFromCommand(cmd)
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Query updated for pool %s.\n", output.PoolID)
	return err
}

func renderPaymentPoolBalances(cmd *cobra.Command, output paymentscmd.GetPoolBalancesOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Balances) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No pool balances found.")
		return err
	}
	for _, balance := range output.Balances {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", balance.Asset, balance.Amount, strings.Join(balance.RelatedAccounts, ",")); err != nil {
			return err
		}
	}
	return nil
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	task := output.Task
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", task.ID); err != nil {
		return err
	}
	if task.ConnectorID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Connector ID\t%s\n", task.ConnectorID); err != nil {
			return err
		}
	}
	if task.CreatedObjectID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created object ID\t%s\n", task.CreatedObjectID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Status\t%s\n", task.Status); err != nil {
		return err
	}
	if task.Error != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Error\t%s\n", task.Error); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", task.CreatedAt.Format(time.RFC3339)); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Updated at\t%s\n", task.UpdatedAt.Format(time.RFC3339))
	return err
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
			rt, err := runtimeFromCommand(cmd)
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Task ID: %s\n", output.TaskID); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Transfer initiation created with ID: %s\n", output.TransferInitiationID)
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.TransferInitiations) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No transfer initiations found.")
		return err
	}
	for _, transfer := range output.TransferInitiations {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			transfer.ID,
			transfer.Type,
			transfer.Amount,
			transfer.Asset,
			transfer.Status,
			transfer.CreatedAt.Format(time.RFC3339),
		); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderPaymentTransferInitiation(cmd *cobra.Command, output paymentscmd.GetTransferInitiationOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	transfer := output.TransferInitiation
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", transfer.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Reference\t%s\n", transfer.Reference); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Amount\t%s\n", transfer.Amount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Asset\t%s\n", transfer.Asset); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Status\t%s\n", transfer.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Connector ID\t%s\n", transfer.ConnectorID); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", transfer.CreatedAt.Format(time.RFC3339))
	return err
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Task ID: %s\n", output.TaskID); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Transfer initiation %s %s.\n", output.TransferInitiationID, done)
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Transfer initiation %s status updated to %s.\n", output.TransferInitiationID, output.Status)
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
			rt, err := runtimeFromCommand(cmd)
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
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if output.TaskID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Task ID: %s\n", output.TaskID); err != nil {
			return err
		}
	}
	if output.ReversalID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Reversal ID: %s\n", output.ReversalID); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Transfer initiation %s reversed.\n", output.TransferInitiationID)
	return err
}
