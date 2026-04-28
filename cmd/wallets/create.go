package wallets

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type CreateStore struct {
	WalletID string `json:"walletId"`
}
type CreateController struct {
	store        *CreateStore
	metadataFlag string
	ikFlag       string
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewDefaultCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store:        NewDefaultCreateStore(),
		metadataFlag: "metadata",
		ikFlag:       "ik",
	}
}

func NewCreateCommand() *cobra.Command {
	c := NewCreateController()
	return fctl.NewCommand("create <name>",
		fctl.WithShortDescription("Create a new wallet"),
		fctl.WithAliases("cr"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringSliceFlag(c.metadataFlag, []string{""}, "Metadata to use"),
		fctl.WithStringFlag(c.ikFlag, "", "Idempotency key"),
		fctl.WithController[*CreateStore](c),
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

	if !fctl.CheckStackApprobation(cmd, "You are about to create a wallet") {
		return nil, fctl.ErrMissingApproval
	}

	metadata, err := fctl.ParseMetadata(fctl.GetStringSlice(cmd, c.metadataFlag))
	if err != nil {
		return nil, err
	}

	request := operations.CreateWalletRequest{
		CreateWalletRequest: &shared.CreateWalletRequest{
			Name:     args[0],
			Metadata: metadata,
		},
		IdempotencyKey: fctl.Ptr(fctl.GetString(cmd, c.ikFlag)),
	}
	response, err := stackClient.Wallets.V1.CreateWallet(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("creating wallet: %w", err)
	}

	c.store.WalletID = response.CreateWalletResponse.Data.ID

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln(
		"Wallet created successfully with ID: %s", c.store.WalletID)
	return nil
}
