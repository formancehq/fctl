package connections

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type UninstallStore struct {
	Name string `json:"name"`
}

type UninstallController struct {
	approve approvalFunc
	store   *UninstallStore
}

var _ fctl.Controller[*UninstallStore] = (*UninstallController)(nil)

func NewUninstallController() *UninstallController {
	return &UninstallController{
		approve: fctl.CheckStackApprobation,
		store:   &UninstallStore{},
	}
}

func NewUninstallCommand() *cobra.Command {
	controller := NewUninstallController()
	return fctl.NewCommand(
		"uninstall <connection>",
		fctl.WithAliases("delete", "remove", "rm", "u"),
		fctl.WithShortDescription("Uninstall a Connectivity connection"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectionNames()),
		fctl.WithConfirmFlag(),
		fctl.WithController[*UninstallStore](controller),
	)
}

func (c *UninstallController) GetStore() *UninstallStore {
	return c.store
}

func (c *UninstallController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	client, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, client.Connectivity.V1)
}

func (c *UninstallController) run(cmd *cobra.Command, args []string, client *sdk.FormanceV1) (fctl.Renderable, error) {
	name := args[0]
	if !c.approve(cmd, "You are about to uninstall Connectivity connection %q", name) {
		return nil, fctl.ErrMissingApproval
	}
	response, err := client.DeleteConnection(cmd.Context(), operations.DeleteConnectionRequest{Name: url.PathEscape(name)}, nil)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("delete connectivity connection %q: empty response", name)
	}
	if err := connectivityinternal.ResponseError(response.StatusCode, response.Error, response.RawResponse); err != nil {
		return nil, err
	}
	c.store.Name = name
	return c, nil
}

func (c *UninstallController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Connection %q uninstalled.\n", c.store.Name)
	return err
}
