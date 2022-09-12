package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	ledgerclient "github.com/numary/numary-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func printTransaction(cmd *cobra.Command, tx ledgerclient.Transaction) {
	if tx.Reference != nil && *tx.Reference != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Reference: %s", *tx.Reference)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Pre commit volumes:")
	for account, v := range *tx.PreCommitVolumes {
		fmt.Fprintf(cmd.OutOrStdout(), "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(cmd.OutOrStdout(), "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Post commit volumes:")
	for account, v := range *tx.PostCommitVolumes {
		fmt.Fprintf(cmd.OutOrStdout(), "\tAddress: %s\r\n", account)
		for asset, volumes := range v {
			fmt.Fprintf(cmd.OutOrStdout(), "\t\tAsset: %s\t\tInput: %f\tOutput: %f\tBalance: %f\r\n",
				asset, volumes.Input, volumes.Output, *volumes.Balance)
		}
	}
	if len(tx.Metadata) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Metadata:")
		for k, v := range tx.Metadata {
			fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", k, v)
		}
	}
}

var transactionsCommand = &cobra.Command{
	Use:   "transactions",
	Short: "Manage transactions (create/read)",
}

var listTransactionsCommand = &cobra.Command{
	Use:   "list",
	Short: "list transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		ledger := viper.GetString(ledgerFlag)
		rsp, _, err := ledgerClient.TransactionsApi.ListTransactions(cmd.Context(), ledger).Execute()
		if err != nil {
			return err
		}
		if len(rsp.Cursor.Data) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No transactions found.")
			return nil
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Transactions: ")
		for _, s := range rsp.Cursor.Data {
			fmt.Fprintf(cmd.OutOrStdout(), "-> Transaction: %d\r\n", s.Txid)
			printTransaction(cmd, s)
		}
		return nil
	},
}

const (
	numscriptVarFlag       = "var"
	numscriptMetadataFlag  = "metadata"
	numscriptReferenceFlag = "reference"
)

var numscriptCommand = &cobra.Command{
	Use:   "num -|[FILENAME]",
	Args:  cobra.ExactArgs(1),
	Short: "execute a numscript script on a ledger",
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		var script string
		if args[0] == "-" {
			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil && err != io.EOF {
				return errors.Wrapf(err, "reading stdin")
			}

			script = string(data)
		} else {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return errors.Wrapf(err, "reading file %s", args[0])
			}
			script = string(data)
		}

		vars := map[string]interface{}{}
		for _, v := range viper.GetStringSlice(numscriptVarFlag) {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 1 {
				return fmt.Errorf("malformed var: %s", v)
			}
			vars[parts[0]] = parts[1]
		}

		metadata := map[string]interface{}{}
		for _, v := range viper.GetStringSlice(numscriptMetadataFlag) {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 1 {
				return fmt.Errorf("malformed metadata: %s", v)
			}
			metadata[parts[0]] = parts[1]
		}
		reference := viper.GetString(numscriptReferenceFlag)

		ledger := viper.GetString(ledgerFlag)
		rsp, _, err := ledgerClient.ScriptApi.
			RunScript(cmd.Context(), ledger).
			Script(ledgerclient.Script{
				Plain:     script,
				Metadata:  metadata,
				Vars:      &vars,
				Reference: &reference,
			}).
			Execute()
		if err != nil {
			return err
		}
		if err != nil {
			return errors.Wrapf(err, "executing numscript")
		}
		if rsp.ErrorCode != nil && *rsp.ErrorCode != "" {
			return errors.New(*rsp.ErrorMessage)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created transaction ID: %d\r\n", rsp.Transaction.Txid)
		printTransaction(cmd, *rsp.Transaction)

		return nil
	},
}

var revertTransactionCommand = &cobra.Command{
	Use:   "revert [TXID]",
	Short: "revert a transaction",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		txId, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return errors.Wrapf(err, "parsing txid")
		}

		ledger := viper.GetString(ledgerFlag)
		rsp, _, err := ledgerClient.TransactionsApi.RevertTransaction(cmd.Context(), ledger, int32(txId)).Execute()
		if err != nil {
			return errors.Wrapf(err, "reverting transaction")
		}
		printTransaction(cmd, rsp.Data)
		return nil
	},
}

var showTransactionCommand = &cobra.Command{
	Use:   "show [TXID]",
	Short: "print a transaction",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ledgerClient, err := getLedgerClient(cmd.Context())
		if err != nil {
			return err
		}

		txId, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return errors.Wrapf(err, "parsing txid")
		}

		ledger := viper.GetString(ledgerFlag)
		rsp, _, err := ledgerClient.TransactionsApi.GetTransaction(cmd.Context(), ledger, int32(txId)).Execute()
		if err != nil {
			return errors.Wrapf(err, "retrieving transaction")
		}
		printTransaction(cmd, rsp.Data)
		return nil
	},
}

func init() {
	transactionsCommand.PersistentFlags().String(stackFlag, "", "Specific stack (not required if only one stack is present)")
	transactionsCommand.PersistentFlags().String(ledgerFlag, "default", "Specific ledger ")
	numscriptCommand.Flags().StringSliceP(numscriptVarFlag, "v", []string{""}, "Variables to use")
	numscriptCommand.Flags().StringSliceP(numscriptMetadataFlag, "m", []string{""}, "Metadata to use")
	numscriptCommand.Flags().StringP(numscriptReferenceFlag, "r", "", "Reference to add to the generated transaction")

	transactionsCommand.AddCommand(listTransactionsCommand, numscriptCommand, revertTransactionCommand, showTransactionCommand)
	rootCommand.AddCommand(transactionsCommand)
}
