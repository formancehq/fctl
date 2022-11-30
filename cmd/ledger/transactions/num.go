package transactions

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/ledger/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	const (
		amountVarFlag  = "amount-var"
		portionVarFlag = "portion-var"
		accountVarFlag = "account-var"
		metadataFlag   = "metadata"
		referenceFlag  = "reference"
	)
	return internal2.NewCommand("num -|[FILENAME]",
		internal2.WithShortDescription("Execute a numscript script on a ledger"),
		internal2.WithDescription(`More help on variables can be found here: https://docs.formance.com/oss/ledger/reference/numscript/variables`),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithStringSliceFlag(amountVarFlag, []string{""}, "Pass a variable of type 'amount'"),
		internal2.WithStringSliceFlag(portionVarFlag, []string{""}, "Pass a variable of type 'portion'"),
		internal2.WithStringSliceFlag(accountVarFlag, []string{""}, "Pass a variable of type 'account'"),
		internal2.WithStringSliceFlag(metadataFlag, []string{""}, "Metadata to use"),
		internal2.WithStringFlag(referenceFlag, "", "Reference to add to the generated transaction"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}
			ledgerClient, err := internal2.NewStackClient(cmd, cfg)
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
			for _, v := range internal2.GetStringSlice(cmd, accountVarFlag) {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 1 {
					return fmt.Errorf("malformed var: %s", v)
				}
				vars[parts[0]] = parts[1]
			}
			for _, v := range internal2.GetStringSlice(cmd, portionVarFlag) {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 1 {
					return fmt.Errorf("malformed var: %s", v)
				}
				vars[parts[0]] = parts[1]
			}
			for _, v := range internal2.GetStringSlice(cmd, amountVarFlag) {
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

			reference := internal2.GetString(cmd, referenceFlag)

			metadata, err := internal.ParseMetadata(internal2.GetStringSlice(cmd, metadataFlag))
			if err != nil {
				return err
			}

			ledger := internal2.GetString(cmd, internal.LedgerFlag)
			response, _, err := ledgerClient.ScriptApi.
				RunScript(cmd.Context(), ledger).
				Script(formance.Script{
					Plain:     script,
					Metadata:  metadata,
					Vars:      vars,
					Reference: &reference,
				}).
				Execute()
			if err != nil {
				return err
			}
			if err != nil {
				return errors.Wrapf(err, "executing numscript")
			}
			if response.ErrorCode != nil && *response.ErrorCode != "" {
				if response.ErrorMessage != nil {
					return errors.New(*response.ErrorMessage)
				}
				return errors.New(*response.ErrorCode)
			}

			return internal.PrintTransaction(cmd.OutOrStdout(), *response.Transaction)
		}),
	)
}
