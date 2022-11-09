package fctl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/numary/ledger/client"
)

func NewLedgerClientFromContext(ctx context.Context, profile *Profile, httpClient *http.Client, organizationID, stackID string) (*client.APIClient, error) {
	token, err := profile.GetStackToken(ctx, httpClient, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	config := client.NewConfiguration()
	config.Servers = client.ServerConfigurations{{
		URL: MustApiUrl(*profile, organizationID, stackID, "ledger").String(),
	}}
	config.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	config.HTTPClient = httpClient

	return client.NewAPIClient(config), nil
}

func PrintLedgerTransaction(out io.Writer, tx client.Transaction) {
	fmt.Fprintf(out, "Date: %s\r\n", tx.Timestamp.Format(time.RFC3339))
	if tx.Reference != nil && *tx.Reference != "" {
		fmt.Fprintf(out, "Reference: %s\r\n", *tx.Reference)
	}
	fmt.Fprintln(out, "Pre commit volumes:")
	for account, v := range *tx.PreCommitVolumes {
		fmt.Fprintf(out, "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(out, "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	fmt.Fprintln(out, "Post commit volumes:")
	for account, v := range *tx.PostCommitVolumes {
		fmt.Fprintf(out, "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(out, "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	if len(tx.Metadata) > 0 {
		fmt.Fprintln(out, "Metadata:")
		for k, v := range tx.Metadata {
			fmt.Fprintf(out, "\t- %s: %s\r\n", k, v)
		}
	}
}
