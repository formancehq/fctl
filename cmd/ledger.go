package cmd

import (
	"context"

	"github.com/numary/fctl/pkg/ledger"
	ledgerclient "github.com/numary/ledger/client"
	"github.com/spf13/viper"
)

const (
	stackFlag  = "stack"
	ledgerFlag = "ledger"
)

func getLedgerClient(ctx context.Context) (*ledgerclient.APIClient, error) {
	organization, err := findOrganizationId(ctx)
	if err != nil {
		return nil, err
	}

	stack, err := findStackId(ctx, organization)
	if err != nil {
		return nil, err
	}

	return ledger.NewClient(currentProfile, viper.GetBool(debugFlag), organization, stack), nil
}
