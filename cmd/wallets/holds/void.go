package holds

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type VoidStore struct {
	Success bool   `json:"success"`
	HoldId  string `json:"holdId"`
}
type VoidController struct {
	store  *VoidStore
	ikFlag string
}

var _ fctl.Controller[*VoidStore] = (*VoidController)(nil)

func NewDefaultVoidStore() *VoidStore {
	return &VoidStore{}
}

func NewVoidController() *VoidController {
	return &VoidController{
		store:  NewDefaultVoidStore(),
		ikFlag: "ik",
	}
}

func NewVoidCommand() *cobra.Command {
	c := NewVoidController()
	return fctl.NewCommand("void <hold-id>",
		fctl.WithShortDescription("Void a hold"),
		fctl.WithAliases("v"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(c.ikFlag, "", "Idempotency Key"),
		fctl.WithController[*VoidStore](c),
	)
}

func (c *VoidController) GetStore() *VoidStore {
	return c.store
}

func (c *VoidController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	request := operations.VoidHoldRequest{
		IdempotencyKey: fctl.Ptr(fctl.GetString(cmd, c.ikFlag)),
		HoldID:         args[0],
	}
	_, err = stackClient.Wallets.V1.VoidHold(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("voiding hold: %w", err)
	}

	c.store.Success = true
	c.store.HoldId = args[0]

	return c, nil
}

func (c *VoidController) Render(cmd *cobra.Command, args []string) error {

	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Hold '%s' voided!", args[0])

	return nil
}
