package cmd

import (
	"context"

	"github.com/numary/fctl/pkg/payments"
	"github.com/numary/payments/client"
	"github.com/spf13/viper"
)

func getPaymentsClient(ctx context.Context) (*client.APIClient, error) {
	organization, err := findOrganizationId(ctx)
	if err != nil {
		return nil, err
	}

	stack, err := findStackId(ctx, organization)
	if err != nil {
		return nil, err
	}

	return payments.NewClient(currentProfile, viper.GetBool(debugFlag), organization, stack), nil
}