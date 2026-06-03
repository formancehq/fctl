package pools

import (
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v4/pkg/models/payments"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UpdateQueryStore struct {
	PoolID string `json:"poolID"`
}

type UpdateQueryController struct {
	PaymentsVersion versions.Version

	store *UpdateQueryStore
}

func (c *UpdateQueryController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*UpdateQueryStore] = (*UpdateQueryController)(nil)

func NewUpdateQueryStore() *UpdateQueryStore {
	return &UpdateQueryStore{}
}

func NewUpdateQueryController() *UpdateQueryController {
	return &UpdateQueryController{
		store: NewUpdateQueryStore(),
	}
}

func NewUpdateQueryCommand() *cobra.Command {
	c := NewUpdateQueryController()
	return fctl.NewCommand("update-query <poolID> <file>|-",
		fctl.WithConfirmFlag(),
		fctl.WithShortDescription("Update the query of a dynamic pool"),
		fctl.WithAliases("uq"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*UpdateQueryStore](c),
	)
}

func (c *UpdateQueryController) GetStore() *UpdateQueryStore {
	return c.store
}

func (c *UpdateQueryController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if !c.PaymentsVersion.IsAtLeast(versions.V3, 1) {
		return nil, fmt.Errorf("update-query is only supported in payments >= v3.1.0")
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to update the query of pool '%s'", args[0]) {
		return nil, fctl.ErrMissingApproval
	}

	script, err := fctl.ReadFile(cmd, args[1])
	if err != nil {
		return nil, err
	}

	var request payments.V3UpdatePoolQueryRequest
	if err := json.Unmarshal([]byte(script), &request); err != nil {
		return nil, err
	}

	response, err := stackClient.Payments.V3.UpdatePoolQuery(cmd.Context(), operations.V3UpdatePoolQueryRequest{
		PoolID:                   args[0],
		V3UpdatePoolQueryRequest: &request,
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.PoolID = args[0]

	return c, nil
}

func (c *UpdateQueryController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Query updated for pool: %s", c.store.PoolID)
	return nil
}
