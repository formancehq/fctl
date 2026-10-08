package connectorinstances

import (
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
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
		"uninstall <connectorinstance>",
		fctl.WithAliases("delete", "remove", "rm", "u"),
		fctl.WithShortDescription("Uninstall a Connectivity connector instance"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorInstanceNames(factory)),
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
	if !c.approve(cmd, "You are about to uninstall Connectivity connector instance %q", name) {
		return nil, fctl.ErrMissingApproval
	}
	if err := client.DeleteConnectorInstance(cmd.Context(), name); err != nil {
		return nil, err
	}
	c.store.Name = name
	return c, nil
}

func (c *UninstallController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Connector instance %q uninstalled.\n", c.store.Name)
	return err
}
