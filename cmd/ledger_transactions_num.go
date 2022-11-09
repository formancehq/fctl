package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/pkg"
	ledgerclient "github.com/numary/ledger/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLedgerTransactionsNumscriptCommand() *cobra.Command {
	const (
		amountVarFlag  = "amount-var"
		portionVarFlag = "portion-var"
		accountVarFlag = "account-var"
		metadataFlag   = "metadata"
		referenceFlag  = "reference"
	)
	return newCommand("num -|[FILENAME]",
		withShortDescription("execute a numscript script on a ledger"),
		withDescription(`More help on variables can be found here: https://docs.formance.com/oss/ledger/reference/numscript/variables`),
		withArgs(cobra.ExactArgs(1)),
		withStringSliceFlag(amountVarFlag, []string{""}, "Pass a variable of type 'amount'"),
		withStringSliceFlag(portionVarFlag, []string{""}, "Pass a variable of type 'portion'"),
		withStringSliceFlag(accountVarFlag, []string{""}, "Pass a variable of type 'account'"),
		withStringSliceFlag(metadataFlag, []string{""}, "Metadata to use"),
		withStringFlag(referenceFlag, "", "Reference to add to the generated transaction"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			ledgerClient, err := newLedgerClient(cmd)
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
			for _, v := range viper.GetStringSlice(accountVarFlag) {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 1 {
					return fmt.Errorf("malformed var: %s", v)
				}
				vars[parts[0]] = parts[1]
			}
			for _, v := range viper.GetStringSlice(portionVarFlag) {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 1 {
					return fmt.Errorf("malformed var: %s", v)
				}
				vars[parts[0]] = parts[1]
			}
			for _, v := range viper.GetStringSlice(amountVarFlag) {
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

			reference := viper.GetString(referenceFlag)

			metadata, err := metadataFromFlag(metadataFlag)
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

			fctl.PrintLedgerTransaction(cmd.OutOrStdout(), *rsp.Transaction)

			return nil
		}),
	)
}
