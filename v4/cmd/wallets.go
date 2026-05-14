package cmd

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/capabilities"
	walletscmd "github.com/formancehq/fctl/v4/internal/commands/wallets"
)

func newWalletsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "wallets",
		Short: "Manage wallets",
	}
	command.AddCommand(newWalletsCreateCommand())
	command.AddCommand(newWalletsCreditCommand())
	command.AddCommand(newWalletsDebitCommand())
	command.AddCommand(newWalletsBalancesCommand())
	command.AddCommand(newWalletsHoldsCommand())
	command.AddCommand(newWalletsTransactionsCommand())
	command.AddCommand(newWalletsListCommand())
	command.AddCommand(newWalletsShowCommand("show", nil, false))
	command.AddCommand(newWalletsShowCommand("get", []string{"g"}, true))
	command.AddCommand(newWalletsUpdateCommand())
	return command
}

func newWalletsTransactionsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "transactions",
		Short: "Manage wallet transactions",
	}
	command.AddCommand(newWalletsTransactionsListCommand())
	return command
}

func newWalletsTransactionsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var walletID string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List wallet transactions",
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
			service := walletscmd.ListTransactionsService{
				Handlers: walletscmd.SDKListTransactionsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureListTransactions,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.ListTransactionsInput{
				PageSize: pageSize,
				Cursor:   cursor,
				WalletID: walletID,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletTransactions(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&walletID, "wallet-id", "", "Filter transactions by wallet ID")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsHoldsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "holds",
		Short: "Manage wallet holds",
	}
	command.AddCommand(newWalletsHoldsListCommand())
	command.AddCommand(newWalletsHoldsShowCommand())
	command.AddCommand(newWalletsHoldsVoidCommand())
	command.AddCommand(newWalletsHoldsConfirmCommand())
	return command
}

func newWalletsHoldsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var walletID string
	var metadata []string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List wallet holds",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := walletscmd.ListHoldsService{
				Handlers: walletscmd.SDKListHoldsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureListHolds,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.ListHoldsInput{
				PageSize: pageSize,
				Cursor:   cursor,
				WalletID: walletID,
				Metadata: parsedMetadata,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletHolds(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&walletID, "wallet-id", "", "Filter holds by wallet ID")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Metadata filter as key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsHoldsShowCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:   "show <hold-id>",
		Short: "Show a wallet hold",
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
			service := walletscmd.GetHoldService{
				Handlers: walletscmd.SDKGetHoldHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureGetHold,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.GetHoldInput{HoldID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletHold(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsHoldsVoidCommand() *cobra.Command {
	var confirm bool
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:   "void <hold-id>",
		Short: "Void a wallet hold",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("wallets holds void requires --confirm")
			}
			if cmd.Flags().Changed("ik") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --ik has been deprecated, use --idempotency-key")
			}
			output, err := runWalletHoldActionCommand(cmd, walletHoldActionCommandRequest{
				Feature:        walletscmd.FeatureVoidHold,
				Handlers:       walletscmd.SDKVoidHoldHandlers,
				HoldID:         args[0],
				IdempotencyKey: idempotencyKey,
				APIVersion:     apiVersion,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletHoldVoided(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm hold void")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&idempotencyKey, "ik", "", "Deprecated alias for --idempotency-key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsHoldsConfirmCommand() *cobra.Command {
	var confirm bool
	var amount string
	var final bool
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:   "confirm <hold-id>",
		Short: "Confirm a wallet hold",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("wallets holds confirm requires --confirm")
			}
			if cmd.Flags().Changed("ik") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --ik has been deprecated, use --idempotency-key")
			}
			parsedAmount, err := parseOptionalBigInt(amount, "amount")
			if err != nil {
				return err
			}
			var finalPtr *bool
			if cmd.Flags().Changed("final") {
				finalPtr = &final
			}
			output, err := runWalletHoldActionCommand(cmd, walletHoldActionCommandRequest{
				Feature:        walletscmd.FeatureConfirmHold,
				Handlers:       walletscmd.SDKConfirmHoldHandlers,
				HoldID:         args[0],
				Amount:         parsedAmount,
				Final:          finalPtr,
				IdempotencyKey: idempotencyKey,
				APIVersion:     apiVersion,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletHoldConfirmed(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm hold confirmation")
	command.Flags().StringVar(&amount, "amount", "", "Amount to confirm")
	command.Flags().BoolVar(&final, "final", false, "Finalize the hold confirmation")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&idempotencyKey, "ik", "", "Deprecated alias for --idempotency-key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

type walletHoldActionCommandRequest struct {
	Feature        capabilities.Feature
	Handlers       func(*formance.Formance) []walletscmd.HoldActionHandler
	HoldID         string
	Amount         *big.Int
	Final          *bool
	IdempotencyKey string
	APIVersion     string
}

func runWalletHoldActionCommand(cmd *cobra.Command, request walletHoldActionCommandRequest) (walletscmd.HoldActionOutput, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return walletscmd.HoldActionOutput{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return walletscmd.HoldActionOutput{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	service := walletscmd.VoidHoldService{
		Handlers: request.Handlers(sdk),
		Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			versionRequest := capabilities.VersionResolutionRequest{
				Product:         walletscmd.ProductWallets,
				Feature:         request.Feature,
				HandlerVersions: handlerVersions,
			}
			if request.APIVersion != "" {
				versionRequest.Policy = capabilities.VersionPolicyPinned
				versionRequest.PinnedVersion = capabilities.APIVersion(request.APIVersion)
			}
			return rt.ResolveAPIVersion(ctx, versionRequest)
		},
	}
	return service.Run(cmd.Context(), walletscmd.HoldActionInput{
		HoldID:         request.HoldID,
		Amount:         request.Amount,
		Final:          request.Final,
		IdempotencyKey: request.IdempotencyKey,
	})
}

func newWalletsBalancesCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "balances",
		Short: "Manage wallet balances",
	}
	command.AddCommand(newWalletsBalancesCreateCommand())
	command.AddCommand(newWalletsBalancesListCommand())
	command.AddCommand(newWalletsBalancesShowCommand())
	return command
}

func newWalletsBalancesCreateCommand() *cobra.Command {
	var confirm bool
	var expiresAt string
	var priority string
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:   "create <wallet-id> <balance-name>",
		Short: "Create a wallet balance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("wallets balances create requires --confirm")
			}
			if cmd.Flags().Changed("ik") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --ik has been deprecated, use --idempotency-key")
			}
			parsedPriority, err := parseOptionalBigInt(priority, "priority")
			if err != nil {
				return err
			}
			parsedExpiresAt, err := parseOptionalRFC3339(expiresAt, "expires-at")
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
			service := walletscmd.CreateBalanceService{
				Handlers: walletscmd.SDKCreateBalanceHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureCreateBalance,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.CreateBalanceInput{
				WalletID:       args[0],
				Name:           args[1],
				Priority:       parsedPriority,
				ExpiresAt:      parsedExpiresAt,
				IdempotencyKey: idempotencyKey,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletBalanceCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm balance creation")
	command.Flags().StringVar(&expiresAt, "expires-at", "", "Balance expiration time as RFC3339")
	command.Flags().StringVar(&priority, "priority", "", "Balance priority")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&idempotencyKey, "ik", "", "Deprecated alias for --idempotency-key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsBalancesListCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     "list <wallet-id>",
		Aliases: []string{"ls", "l"},
		Short:   "List wallet balances",
		Args:    cobra.ExactArgs(1),
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
			service := walletscmd.ListBalancesService{
				Handlers: walletscmd.SDKListBalancesHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureListBalances,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.ListBalancesInput{WalletID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletBalances(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsBalancesShowCommand() *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     "show <wallet-id> <balance-name>",
		Aliases: []string{"get"},
		Short:   "Show a wallet balance",
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
			service := walletscmd.GetBalanceService{
				Handlers: walletscmd.SDKGetBalanceHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureGetBalance,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.GetBalanceInput{WalletID: args[0], BalanceName: args[1]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletBalance(cmd, output)
		},
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsCreditCommand() *cobra.Command {
	var confirm bool
	var amount string
	var asset string
	var balance string
	var metadata []string
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:     "credit <wallet-id>",
		Aliases: []string{"cr"},
		Short:   "Credit a wallet",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("wallets credit requires --confirm")
			}
			if cmd.Flags().Changed("ik") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --ik has been deprecated, use --idempotency-key")
			}
			parsedAmount, err := parseBigAmount(amount)
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			output, err := runWalletMovementCommand(cmd, walletMovementCommandRequest{
				Feature:        walletscmd.FeatureCreditWallet,
				Handlers:       walletscmd.SDKCreditWalletHandlers,
				WalletID:       args[0],
				Amount:         parsedAmount,
				Asset:          asset,
				Balance:        balance,
				Metadata:       parsedMetadata,
				IdempotencyKey: idempotencyKey,
				APIVersion:     apiVersion,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletCredited(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm wallet credit")
	command.Flags().StringVar(&amount, "amount", "", "Amount to credit")
	command.Flags().StringVar(&asset, "asset", "", "Asset to credit")
	command.Flags().StringVar(&balance, "balance", "", "Balance to credit")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Metadata as key=value")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&idempotencyKey, "ik", "", "Deprecated alias for --idempotency-key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsDebitCommand() *cobra.Command {
	var confirm bool
	var amount string
	var asset string
	var balance string
	var description string
	var metadata []string
	var pending bool
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:     "debit <wallet-id>",
		Aliases: []string{"deb"},
		Short:   "Debit a wallet",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("wallets debit requires --confirm")
			}
			if cmd.Flags().Changed("ik") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --ik has been deprecated, use --idempotency-key")
			}
			parsedAmount, err := parseBigAmount(amount)
			if err != nil {
				return err
			}
			parsedMetadata, err := parseMetadataFlags(metadata)
			if err != nil {
				return err
			}
			output, err := runWalletMovementCommand(cmd, walletMovementCommandRequest{
				Feature:        walletscmd.FeatureDebitWallet,
				Handlers:       walletscmd.SDKDebitWalletHandlers,
				WalletID:       args[0],
				Amount:         parsedAmount,
				Asset:          asset,
				Balance:        balance,
				Description:    description,
				Metadata:       parsedMetadata,
				Pending:        pending,
				IdempotencyKey: idempotencyKey,
				APIVersion:     apiVersion,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletDebited(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm wallet debit")
	command.Flags().StringVar(&amount, "amount", "", "Amount to debit")
	command.Flags().StringVar(&asset, "asset", "", "Asset to debit")
	command.Flags().StringVar(&balance, "balance", "", "Balance to debit")
	command.Flags().StringVar(&description, "description", "", "Debit description")
	command.Flags().BoolVar(&pending, "pending", false, "Create a pending hold")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Metadata as key=value")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&idempotencyKey, "ik", "", "Deprecated alias for --idempotency-key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

type walletMovementCommandRequest struct {
	Feature        capabilities.Feature
	Handlers       func(*formance.Formance) []walletscmd.WalletMovementHandler
	WalletID       string
	Amount         *big.Int
	Asset          string
	Balance        string
	Description    string
	Metadata       map[string]string
	Pending        bool
	IdempotencyKey string
	APIVersion     string
}

func runWalletMovementCommand(cmd *cobra.Command, request walletMovementCommandRequest) (walletscmd.WalletMovementOutput, error) {
	rt, err := runtimeFromCommand(cmd)
	if err != nil {
		return walletscmd.WalletMovementOutput{}, err
	}
	httpClient, err := rt.HTTPClient(cmd.Context())
	if err != nil {
		return walletscmd.WalletMovementOutput{}, err
	}
	sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
	service := walletscmd.CreditWalletService{
		Handlers: request.Handlers(sdk),
		Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
			versionRequest := capabilities.VersionResolutionRequest{
				Product:         walletscmd.ProductWallets,
				Feature:         request.Feature,
				HandlerVersions: handlerVersions,
			}
			if request.APIVersion != "" {
				versionRequest.Policy = capabilities.VersionPolicyPinned
				versionRequest.PinnedVersion = capabilities.APIVersion(request.APIVersion)
			}
			return rt.ResolveAPIVersion(ctx, versionRequest)
		},
	}
	return service.Run(cmd.Context(), walletscmd.WalletMovementInput{
		WalletID:       request.WalletID,
		Amount:         request.Amount,
		Asset:          request.Asset,
		Balance:        request.Balance,
		Description:    request.Description,
		Metadata:       request.Metadata,
		Pending:        request.Pending,
		IdempotencyKey: request.IdempotencyKey,
	})
}

func newWalletsCreateCommand() *cobra.Command {
	var confirm bool
	var metadata []string
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"cr"},
		Short:   "Create a wallet",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("wallets create requires --confirm")
			}
			if cmd.Flags().Changed("ik") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --ik has been deprecated, use --idempotency-key")
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := walletscmd.CreateWalletService{
				Handlers: walletscmd.SDKCreateWalletHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureCreateWallet,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.CreateWalletInput{
				Name:           args[0],
				Metadata:       parsedMetadata,
				IdempotencyKey: idempotencyKey,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletCreated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm wallet creation")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Metadata as key=value")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&idempotencyKey, "ik", "", "Deprecated alias for --idempotency-key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsListCommand() *cobra.Command {
	var pageSize int64 = 15
	var cursor string
	var name string
	var metadata []string
	var apiVersion string

	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List wallets",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := walletscmd.ListWalletsService{
				Handlers: walletscmd.SDKListWalletsHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureListWallets,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.ListWalletsInput{
				PageSize: pageSize,
				Cursor:   cursor,
				Name:     name,
				Metadata: parsedMetadata,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWallets(cmd, output)
		},
	}
	command.Flags().Int64Var(&pageSize, "page-size", 15, "Page size")
	command.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	command.Flags().StringVar(&name, "name", "", "Filter wallets by name")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Metadata filter as key=value")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsShowCommand(use string, aliases []string, deprecated bool) *cobra.Command {
	var apiVersion string

	command := &cobra.Command{
		Use:     use + " <wallet-id>",
		Aliases: aliases,
		Short:   "Show a wallet",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deprecated {
				fmt.Fprintln(cmd.ErrOrStderr(), "Command wallets get has been deprecated, use wallets show")
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
			service := walletscmd.GetWalletService{
				Handlers: walletscmd.SDKGetWalletHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureGetWallet,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.GetWalletInput{WalletID: args[0]})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWallet(cmd, output)
		},
	}
	if deprecated {
		command.Deprecated = "use wallets show"
	}
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func newWalletsUpdateCommand() *cobra.Command {
	var confirm bool
	var metadata []string
	var idempotencyKey string
	var apiVersion string

	command := &cobra.Command{
		Use:     "update <wallet-id>",
		Aliases: []string{"up"},
		Short:   "Update a wallet",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("wallets update requires --confirm")
			}
			if cmd.Flags().Changed("ik") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Flag --ik has been deprecated, use --idempotency-key")
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
			sdk := formance.New(formance.WithServerURL(rt.Target.URL), formance.WithClient(httpClient))
			service := walletscmd.UpdateWalletService{
				Handlers: walletscmd.SDKUpdateWalletHandlers(sdk),
				Resolve: func(ctx context.Context, handlerVersions []capabilities.APIVersion) (capabilities.APIVersion, error) {
					request := capabilities.VersionResolutionRequest{
						Product:         walletscmd.ProductWallets,
						Feature:         walletscmd.FeatureUpdateWallet,
						HandlerVersions: handlerVersions,
					}
					if apiVersion != "" {
						request.Policy = capabilities.VersionPolicyPinned
						request.PinnedVersion = capabilities.APIVersion(apiVersion)
					}
					return rt.ResolveAPIVersion(ctx, request)
				},
			}
			output, err := service.Run(cmd.Context(), walletscmd.UpdateWalletInput{
				WalletID:       args[0],
				Metadata:       parsedMetadata,
				IdempotencyKey: idempotencyKey,
			})
			if err != nil {
				return err
			}
			if handled, err := writeStructuredOutput(cmd, output); handled || err != nil {
				return err
			}
			return renderWalletUpdated(cmd, output)
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm wallet update")
	command.Flags().StringArrayVar(&metadata, "metadata", nil, "Metadata as key=value")
	command.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")
	command.Flags().StringVar(&idempotencyKey, "ik", "", "Deprecated alias for --idempotency-key")
	command.Flags().StringVar(&apiVersion, "api-version", "", "Pin wallets API version")
	return command
}

func parseBigAmount(value string) (*big.Int, error) {
	if value == "" {
		return nil, fmt.Errorf("amount is required")
	}
	amount, ok := big.NewInt(0).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("amount must be an integer")
	}
	return amount, nil
}

func parseOptionalBigInt(value string, name string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	parsed, ok := big.NewInt(0).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("%s must be an integer", name)
	}
	return parsed, nil
}

func renderWalletCreated(cmd *cobra.Command, output walletscmd.CreateWalletOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Wallet created with ID: %s\n", output.WalletID)
	return err
}

func renderWalletBalanceCreated(cmd *cobra.Command, output walletscmd.CreateBalanceOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Balance %s created on wallet %s.\n", output.BalanceName, output.WalletID)
	return err
}

func renderWalletBalances(cmd *cobra.Command, output walletscmd.ListBalancesOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Balances) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No balances found.")
		return err
	}
	for _, balance := range output.Balances {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", balance.Name, balance.Priority); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderWalletBalance(cmd *cobra.Command, output walletscmd.GetBalanceOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	balance := output.Balance
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", balance.Name); err != nil {
		return err
	}
	if balance.Priority != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Priority\t%s\n", balance.Priority); err != nil {
			return err
		}
	}
	if balance.ExpiresAt != nil {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Expires at\t%s\n", balance.ExpiresAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if len(balance.Assets) > 0 {
		assets := make([]string, 0, len(balance.Assets))
		for asset := range balance.Assets {
			assets = append(assets, asset)
		}
		sort.Strings(assets)
		for _, asset := range assets {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Asset\t%s\t%s\n", asset, balance.Assets[asset].String()); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderWalletHolds(cmd *cobra.Command, output walletscmd.ListHoldsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Holds) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No holds found.")
		return err
	}
	for _, hold := range output.Holds {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", hold.ID, hold.WalletID, hold.Asset); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderWalletHold(cmd *cobra.Command, output walletscmd.GetHoldOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	hold := output.Hold
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", hold.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Wallet ID\t%s\n", hold.WalletID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Asset\t%s\n", hold.Asset); err != nil {
		return err
	}
	if hold.OriginalAmount != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Original amount\t%s\n", hold.OriginalAmount); err != nil {
			return err
		}
	}
	if hold.Remaining != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Remaining\t%s\n", hold.Remaining); err != nil {
			return err
		}
	}
	return nil
}

func renderWalletHoldVoided(cmd *cobra.Command, output walletscmd.HoldActionOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Hold %s voided.\n", output.HoldID)
	return err
}

func renderWalletHoldConfirmed(cmd *cobra.Command, output walletscmd.HoldActionOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Hold %s confirmed.\n", output.HoldID)
	return err
}

func renderWalletTransactions(cmd *cobra.Command, output walletscmd.ListTransactionsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Transactions) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No transactions found.")
		return err
	}
	for _, transaction := range output.Transactions {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"%d\t%s\t%s\t%s\n",
			transaction.ID,
			transaction.Timestamp.Format(time.RFC3339),
			transaction.Ledger,
			transaction.Reference,
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

func renderWalletCredited(cmd *cobra.Command, output walletscmd.WalletMovementOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Wallet %s credited.\n", output.WalletID)
	return err
}

func renderWalletDebited(cmd *cobra.Command, output walletscmd.WalletMovementOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if output.HoldID != "" {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Hold ID: %s\n", output.HoldID); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Wallet %s debited.\n", output.WalletID)
	return err
}

func renderWallets(cmd *cobra.Command, output walletscmd.ListWalletsOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	if len(output.Wallets) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No wallets found.")
		return err
	}
	for _, wallet := range output.Wallets {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", wallet.ID, wallet.Name, wallet.Ledger, wallet.CreatedAt.Format(time.RFC3339)); err != nil {
			return err
		}
	}
	if output.HasMore && output.Next != nil {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Next: %s\n", *output.Next)
		return err
	}
	return nil
}

func renderWallet(cmd *cobra.Command, output walletscmd.GetWalletOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	wallet := output.Wallet
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "ID\t%s\n", wallet.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Name\t%s\n", wallet.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Ledger\t%s\n", wallet.Ledger); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Created at\t%s\n", wallet.CreatedAt.Format(time.RFC3339))
	return err
}

func renderWalletUpdated(cmd *cobra.Command, output walletscmd.UpdateWalletOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Wallet %s updated.\n", output.WalletID)
	return err
}
