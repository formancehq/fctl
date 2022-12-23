package wallets

import (
	"fmt"
	"io"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return fctl.NewCommand("show ID",
		fctl.WithShortDescription("Show a wallets"),
		fctl.WithAliases("sh"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return errors.Wrap(err, "fctl.GetConfig")
			}

			organizationID, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := fctl.ResolveStack(cmd, cfg, organizationID)
			if err != nil {
				return err
			}

			client, err := fctl.NewStackClient(cmd, cfg, stack)
			if err != nil {
				return errors.Wrap(err, "creating stack client")
			}

			res, _, err := client.WalletsApi.GetWallet(cmd.Context(), args[0]).Execute()
			if err != nil {
				return errors.Wrap(err, "Creating wallets")
			}

			return PrintWallet(cmd.OutOrStdout(), res.Data)
		}),
	)
}

func PrintWallet(out io.Writer, wallet formance.WalletWithBalances) error {
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), fmt.Sprint(wallet.Id)})
	tableData = append(tableData, []string{pterm.LightCyan("Name"), wallet.Name})

	if err := pterm.DefaultTable.
		WithWriter(out).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	tableData = pterm.TableData{}
	tableData = append(tableData, []string{"Asset", "Amount"})
	for asset, amount := range wallet.Balances {
		tableData = append(tableData, []string{asset, fmt.Sprint(amount)})
	}
	if err := pterm.DefaultTable.
		WithHasHeader(true).
		WithWriter(out).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return nil
}
