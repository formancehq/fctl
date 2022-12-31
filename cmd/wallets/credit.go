package wallets

import (
	"strconv"

	"github.com/formancehq/fctl/cmd/wallets/internal"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCreditWalletCommand() *cobra.Command {
	const (
		metadataFlag = "metadata"
	)
	return fctl.NewCommand("credit [<wallet-id> | --name=<wallet-name>] <amount> <asset>",
		fctl.WithShortDescription("Credit a wallets"),
		fctl.WithAliases("cr"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.RangeArgs(2, 3)),
		fctl.WithStringSliceFlag(metadataFlag, []string{""}, "Metadata to use"),
		internal.WithTargetingWalletByName(),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return errors.Wrap(err, "reading config")
			}

			organizationID, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			stack, err := fctl.ResolveStack(cmd, cfg, organizationID)
			if err != nil {
				return err
			}

			if !fctl.CheckStackApprobation(cmd, stack, "You are about to credit a wallets") {
				return fctl.ErrMissingApproval
			}

			client, err := fctl.NewStackClient(cmd, cfg, stack)
			if err != nil {
				return errors.Wrap(err, "creating stack client")
			}

			var (
				amountStr string
				asset     string
				walletID  string
			)
			switch len(args) {
			case 2:
				amountStr = args[0]
				asset = args[1]
				walletID, err = internal.RetrieveWalletIDFromName(cmd, client)
				if err != nil {
					return err
				}
			case 3:
				walletID = args[0]
				amountStr = args[1]
				asset = args[2]
			}

			amount, err := strconv.ParseInt(amountStr, 10, 32)
			if err != nil {
				return errors.Wrap(err, "parsing amount")
			}

			metadata, err := fctl.ParseMetadata(fctl.GetStringSlice(cmd, metadataFlag))
			if err != nil {
				return err
			}

			_, err = client.WalletsApi.CreditWallet(cmd.Context(), walletID).CreditWalletRequest(formance.CreditWalletRequest{
				Amount: formance.Monetary{
					Asset:  asset,
					Amount: amount,
				},
				Metadata: metadata,
			}).Execute()
			if err != nil {
				return errors.Wrap(err, "Crediting wallets")
			}

			fctl.Success(cmd.OutOrStdout(), "Wallet credited successfully!")

			return nil
		}),
	)
}
