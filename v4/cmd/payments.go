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
