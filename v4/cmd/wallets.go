package cmd

import (
	"context"
	"fmt"
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
	command.AddCommand(newWalletsListCommand())
	command.AddCommand(newWalletsShowCommand("show", nil, false))
	command.AddCommand(newWalletsShowCommand("get", []string{"g"}, true))
	command.AddCommand(newWalletsUpdateCommand())
	return command
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

func renderWalletCreated(cmd *cobra.Command, output walletscmd.CreateWalletOutput) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "API version: %s\n", output.APIVersion); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Wallet created with ID: %s\n", output.WalletID)
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
