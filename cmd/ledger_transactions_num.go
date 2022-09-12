package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	ledgerclient "github.com/numary/ledger/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	numscriptAmountVarFlag  = "amount-var"
	numscriptPortionVarFlag = "portion-var"
	numscriptAccountVarFlag = "account-var"
	numscriptMetadataFlag   = "metadata"
	numscriptReferenceFlag  = "reference"
)

var numscriptCommand = &cobra.Command{
	Use:   "num -|[FILENAME]",
	Args:  cobra.ExactArgs(1),
	Short: "execute a numscript script on a ledger",
	Long:  `More help on variables can be found here: https://docs.formance.com/oss/ledger/reference/numscript/variables`,
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
		for _, v := range viper.GetStringSlice(numscriptAccountVarFlag) {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 1 {
				return fmt.Errorf("malformed var: %s", v)
			}
			vars[parts[0]] = parts[1]
		}
		for _, v := range viper.GetStringSlice(numscriptPortionVarFlag) {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 1 {
				return fmt.Errorf("malformed var: %s", v)
			}
			vars[parts[0]] = parts[1]
		}
		for _, v := range viper.GetStringSlice(numscriptAmountVarFlag) {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 1 {
				return fmt.Errorf("malformed var: %s", v)
			}

			amountParts := strings.SplitN(parts[1], "/", 2)
			if len(amountParts) != 2 {
				return fmt.Errorf("malformed var: %s", v)
			}

			amount, err := strconv.ParseInt(amountParts[0], 10, 64)
			if err != nil {
				return fmt.Errorf("malformed var: %s", v)
			}

			vars[parts[0]] = map[string]any{
				"amount": amount,
				"asset":  amountParts[1],
			}
		}

		reference := viper.GetString(numscriptReferenceFlag)

		metadata, err := metadataFromFlag(numscriptMetadataFlag)
		if err != nil {
			return err
		}

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
			if rsp.ErrorMessage != nil {
				return errors.New(*rsp.ErrorMessage)
			}
			return errors.New(*rsp.ErrorCode)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created transaction ID: %d\r\n", rsp.Transaction.Txid)
		printTransaction(cmd, *rsp.Transaction)

		return nil
	},
}

func init() {
	transactionsCommand.AddCommand(numscriptCommand)
	numscriptCommand.Flags().StringSlice(numscriptAmountVarFlag, []string{""}, "Pass a variable of type 'amount'")
	numscriptCommand.Flags().StringSlice(numscriptPortionVarFlag, []string{""}, "Pass a variable of type 'portion'")
	numscriptCommand.Flags().StringSlice(numscriptAccountVarFlag, []string{""}, "Pass a variable of type 'account'")
	numscriptCommand.Flags().StringSlice(numscriptMetadataFlag, []string{""}, "Metadata to use")
	numscriptCommand.Flags().StringP(numscriptReferenceFlag, "r", "", "Reference to add to the generated transaction")
}
