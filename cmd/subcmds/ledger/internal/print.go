package internal

import (
	"fmt"
	"io"
	"time"

	"github.com/numary/ledger/client"
)

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
