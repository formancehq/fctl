package internal

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	formance "github.com/formancehq/formance-sdk-go/v4"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/wallets"
)

func ParseSubject(subject string, cmd *cobra.Command, client *formance.Formance) (*wallets.Subject, error) {
	var err error
	switch {
	case strings.HasPrefix(subject, "wallet="):
		walletDefinition := strings.TrimPrefix(subject, "wallet=")
		parts := strings.SplitN(walletDefinition, "/", 2)
		balance := "main"
		if len(parts) > 1 {
			balance = parts[1]
		}

		var walletID string
		switch {
		case strings.HasPrefix(walletDefinition, "id:"):
			walletID = strings.TrimPrefix(parts[0], "id:")
		case strings.HasPrefix(walletDefinition, "name:"):
			walletID, err = DiscoverWalletIDFromName(cmd, client, strings.TrimPrefix(parts[0], "name:"))
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("malformed wallet source definition")
		}
		subject := wallets.CreateSubjectWalletSubject(wallets.WalletSubject{
			Identifier: walletID,
			Balance:    &balance,
		})
		return &subject, nil
	case strings.HasPrefix(subject, "account="):
		subject := wallets.CreateSubjectLedgerAccountSubject(wallets.LedgerAccountSubject{
			Identifier: strings.TrimPrefix(subject, "account="),
		})
		return &subject, nil
	default:
		return nil, errors.New("malformed source definition")
	}
}
