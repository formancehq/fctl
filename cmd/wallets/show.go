package wallets

import (
	"github.com/formancehq/fctl/cmd/wallets/internal"
	"github.com/formancehq/fctl/cmd/wallets/internal/views"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ShowStore struct {
	Wallet shared.WalletWithBalances `json:"wallet"`
}
type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewDefaultShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewDefaultShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	c := NewShowController()
	return fctl.NewCommand("show",
		fctl.WithShortDescription("Show a wallets"),
		fctl.WithAliases("sh"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		internal.WithTargetingWalletByID(),
		internal.WithTargetingWalletByName(),
		fctl.WithController[*ShowStore](c),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	walletID, err := internal.RetrieveWalletID(cmd, stackClient)
	if err != nil {
		return nil, err
	}
	if walletID == "" {
		return nil, errors.New("You need to specify wallet id using --id or --name flags")
	}

	response, err := stackClient.Wallets.V1.GetWallet(cmd.Context(), operations.GetWalletRequest{
		ID: walletID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting wallet")
	}

	c.store.Wallet = response.ActivityGetWalletOutput.Data

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	return views.PrintWalletWithMetadata(cmd.OutOrStdout(), c.store.Wallet)
}
