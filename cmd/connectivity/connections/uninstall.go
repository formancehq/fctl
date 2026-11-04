package connections

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type UninstallStore struct {
	Name string `json:"name"`
}

type UninstallController struct {
	factory connectivityinternal.ClientFactory
	approve approvalFunc
	store   *UninstallStore
}

var _ fctl.Controller[*UninstallStore] = (*UninstallController)(nil)

func NewUninstallController(factory connectivityinternal.ClientFactory) *UninstallController {
	return &UninstallController{
		factory: factory,
		approve: fctl.CheckStackApprobation,
		store:   &UninstallStore{},
	}
}

func NewUninstallCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewUninstallController(factory)
	return fctl.NewCommand(
		"uninstall <connection>",
		fctl.WithAliases("delete", "remove", "rm", "u"),
		fctl.WithShortDescription("Uninstall a Connectivity connection"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectionNames(factory)),
		fctl.WithConfirmFlag(),
		fctl.WithController[*UninstallStore](controller),
	)
}

func (c *UninstallController) GetStore() *UninstallStore {
	return c.store
}

func (c *UninstallController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}
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
	if err := connectivityinternal.ResponseError(response.StatusCode, response.Error); err != nil {
		return nil, err
	}
	c.store.Name = name
	return c, nil
}

func (c *UninstallController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Connection %q uninstalled.\n", c.store.Name)
	return err
}
