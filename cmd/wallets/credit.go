package wallets

import (
	"strconv"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCreditWalletCommand() *cobra.Command {
	const (
		metadataFlag = "metadata"
	)
	return fctl.NewCommand("credit ID <amount> <asset>",
		fctl.WithShortDescription("Credit a wallets"),
		fctl.WithAliases("cr"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(3)),
		fctl.WithStringSliceFlag(metadataFlag, []string{""}, "Metadata to use"),
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

			if !fctl.CheckStackApprobation(cmd, stack, "You are about to credit a wallets") {
				return fctl.ErrMissingApproval
			}

			client, err := fctl.NewStackClient(cmd, cfg, stack)
			if err != nil {
				return errors.Wrap(err, "creating stack client")
			}

			amount, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return errors.Wrap(err, "parsing amount")
			}

			metadata, err := fctl.ParseMetadata(fctl.GetStringSlice(cmd, metadataFlag))
			if err != nil {
				return err
			}

			_, err = client.WalletsApi.CreditWallet(cmd.Context(), args[0]).CreditWalletRequest(formance.CreditWalletRequest{
				Amount: formance.Monetary{
					Asset:  args[2],
					Amount: float32(amount),
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
