package internal

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	fctl "github.com/formancehq/fctl/pkg"
)

const (
	walletNameFlag = "name"
	walletIDFlag   = "id"
)

var (
	ErrUndefinedName = errors.New("missing wallet name")
)

func WithTargetingWalletByName() fctl.CommandOptionFn {
	return fctl.WithStringFlag(walletNameFlag, "", "Wallet name to use")
}

func WithTargetingWalletByID() fctl.CommandOptionFn {
	return fctl.WithStringFlag(walletIDFlag, "", "Wallet ID to use")
}

func DiscoverWalletIDFromName(cmd *cobra.Command, client *formance.Formance, walletName string) (string, error) {
	request := operations.ListWalletsRequest{
		Name: &walletName,
	}
	wallets, err := client.Wallets.V1.ListWallets(cmd.Context(), request)
	if err != nil {
		return "", fmt.Errorf("listing wallets to retrieve wallet by name: %w", err)
	}

	if wallets.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status code: %d", wallets.StatusCode)
	}

	if len(wallets.ListWalletsResponse.Cursor.Data) > 1 {
		return "", fmt.Errorf("found multiple wallets with name: %s", walletName)
	}
	if len(wallets.ListWalletsResponse.Cursor.Data) == 0 {
		return "", fmt.Errorf("wallet with name '%s' not found", walletName)
	}
	return wallets.ListWalletsResponse.Cursor.Data[0].ID, nil
}

func RetrieveWalletIDFromName(cmd *cobra.Command, client *formance.Formance) (string, error) {
	walletName := fctl.GetString(cmd, walletNameFlag)
	if walletName == "" {
		return "", ErrUndefinedName
	}
	return DiscoverWalletIDFromName(cmd, client, walletName)
}

func RetrieveWalletID(cmd *cobra.Command, client *formance.Formance) (string, error) {
	walletID, err := RetrieveWalletIDFromName(cmd, client)
	if err != nil && err != ErrUndefinedName {
		return "", err
	}
	if err == ErrUndefinedName {
		return fctl.GetString(cmd, walletIDFlag), nil
	}
	return walletID, nil
}

func RequireWalletID(cmd *cobra.Command, client *formance.Formance) (string, error) {
	walletID, err := RetrieveWalletID(cmd, client)
	if err != nil {
		return "", err
	}
	if walletID == "" {
		return "", errors.New("You need to specify wallet id using --id or --name flags")
	}
	return walletID, nil
}
