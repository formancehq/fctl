package balances

import (
	"fmt"
	"math/big"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/wallets/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateStore struct {
	BalanceName string `json:"balanceName"`
}
type CreateController struct {
	store *CreateStore
}

const expiresAtFlag = "expires-at"

const priorityFlag = "priority"

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewDefaultCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	return fctl.NewCommand("create <balance-name>",
		fctl.WithShortDescription("Create a new balance"),
		fctl.WithAliases("c", "cr"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		internal.WithTargetingWalletByID(),
		internal.WithTargetingWalletByName(),
		fctl.WithStringFlag(expiresAtFlag, "", "Balance expiration date"),
		fctl.WithIntFlag(priorityFlag, 0, "Balance priority"),
		fctl.WithController[*CreateStore](NewCreateController()),
	)
}

func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	walletID, err := internal.RequireWalletID(cmd, stackClient)
	if err != nil {
		return nil, err
	}

	expiresAt, err := fctl.GetDateTime(cmd, expiresAtFlag)
	if err != nil {
		return nil, err
	}

	var priority *big.Int = nil
	priorityInt := fctl.GetInt(cmd, priorityFlag)
	if priorityInt != 0 {
		priority = big.NewInt(int64(priorityInt))
	}

	request := operations.CreateBalanceRequest{
		ID: walletID,
		CreateBalanceRequest: &shared.CreateBalanceRequest{
			Name:      args[0],
			ExpiresAt: expiresAt,
			Priority:  priority,
		},
	}
	response, err := stackClient.Wallets.V1.CreateBalance(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("creating balance: %w", err)
	}

	c.store.BalanceName = response.CreateBalanceResponse.Data.Name
	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln(
		"Balance created successfully with name: %s", c.store.BalanceName)
	return nil

}
