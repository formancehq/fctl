package transactions

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/ledger/internal"
	ledgerclient "github.com/numary/ledger/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewLedgerTransactionsNumscriptCommand() *cobra.Command {
	const (
		amountVarFlag  = "amount-var"
		portionVarFlag = "portion-var"
		accountVarFlag = "account-var"
		metadataFlag   = "metadata"
		referenceFlag  = "reference"
	)
	return cmdbuilder.NewCommand("num -|[FILENAME]",
		cmdbuilder.WithShortDescription("execute a numscript script on a ledger"),
		cmdbuilder.WithDescription(`More help on variables can be found here: https://docs.formance.com/oss/ledger/reference/numscript/variables`),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithStringSliceFlag(amountVarFlag, []string{""}, "Pass a variable of type 'amount'"),
		cmdbuilder.WithStringSliceFlag(portionVarFlag, []string{""}, "Pass a variable of type 'portion'"),
		cmdbuilder.WithStringSliceFlag(accountVarFlag, []string{""}, "Pass a variable of type 'account'"),
		cmdbuilder.WithStringSliceFlag(metadataFlag, []string{""}, "Metadata to use"),
		cmdbuilder.WithStringFlag(referenceFlag, "", "Reference to add to the generated transaction"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			ledgerClient, err := internal.NewLedgerClient(cmd, cfg)
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

			metadata := map[string]interface{}{}
			for _, v := range viper.GetStringSlice(metadataFlag) {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 1 {
					return fmt.Errorf("malformed metadata: %s", v)
				}
				metadata[parts[0]] = parts[1]
			}

			ledger := viper.GetString(internal.LedgerFlag)
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

			internal.PrintLedgerTransaction(cmd.OutOrStdout(), *rsp.Transaction)

			return nil
		}),
	)
}
