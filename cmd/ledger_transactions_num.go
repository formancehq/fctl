package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	ledgerclient "github.com/numary/ledger/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
			return errors.New(*rsp.ErrorMessage)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created transaction ID: %d\r\n", rsp.Transaction.Txid)
		printTransaction(cmd, *rsp.Transaction)

		return nil
	},
}

func init() {
	transactionsCommand.AddCommand(numscriptCommand)
	numscriptCommand.Flags().StringSliceP(numscriptVarFlag, "v", []string{""}, "Variables to use")
	numscriptCommand.Flags().StringSliceP(numscriptMetadataFlag, "m", []string{""}, "Metadata to use")
	numscriptCommand.Flags().StringP(numscriptReferenceFlag, "r", "", "Reference to add to the generated transaction")
}
