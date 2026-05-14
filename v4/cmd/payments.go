package cmd

import (
	"context"
	"fmt"
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
