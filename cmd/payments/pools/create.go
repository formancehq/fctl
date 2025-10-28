package pools

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
)

type CreateStore struct {
	PoolID string `json:"poolID"`
}
type CreateController struct {
	PaymentsVersion versions.Version

	store *CreateStore
}

func (c *CreateController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*CreateStore] = (*CreateController)(nil)

func NewCreateStore() *CreateStore {
	return &CreateStore{}
}

func NewCreateController() *CreateController {
	return &CreateController{
		store: NewCreateStore(),
	}
}

func NewCreateCommand() *cobra.Command {
	c := NewCreateController()
	return fctl.NewCommand("create <file>|-",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Create a pool"),
		fctl.WithAliases("cr", "c"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithController[*CreateStore](c),
	)
}

func (c *CreateController) GetStore() *CreateStore {
	return c.store
}

func (c *CreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}
	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V1 {
		return nil, fmt.Errorf("pools are only supported in >= v1.0.0")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to create a new pool") {
		return nil, fctl.ErrMissingApproval
	}

	script, err := fctl.ReadFile(cmd, args[0])
	if err != nil {
		return nil, err
	}

	request := shared.PoolRequest{}
	if err := json.Unmarshal([]byte(script), &request); err != nil {
		return nil, err
	}

	//nolint:gosimple
	response, err := stackClient.Payments.V1.CreatePool(cmd.Context(), request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.PoolID = response.PoolResponse.Data.ID

	return c, nil
}

func (c *CreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Pool created with ID: %s", c.store.PoolID)

	return nil
}
