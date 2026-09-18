package connectorinstances

import (
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type SuspensionStore struct {
	Name    string `json:"name"`
	Suspend bool   `json:"suspend"`
}

type SuspensionController struct {
	factory connectivityinternal.ClientFactory
	approve approvalFunc
	store   *SuspensionStore
	suspend bool
}

var _ fctl.Controller[*SuspensionStore] = (*SuspensionController)(nil)

func NewSuspensionController(factory connectivityinternal.ClientFactory, suspend bool) *SuspensionController {
	return &SuspensionController{
		factory: factory,
		approve: fctl.CheckStackApprobation,
		store:   &SuspensionStore{},
		suspend: suspend,
	}
}

func NewSuspendCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	return newSuspensionCommand(factory, "suspend", "Suspend a Connectivity connector instance", true)
}

func NewUnsuspendCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	return newSuspensionCommand(factory, "unsuspend", "Resume a Connectivity connector instance", false)
}

func newSuspensionCommand(factory connectivityinternal.ClientFactory, name, description string, suspend bool) *cobra.Command {
	controller := NewSuspensionController(factory, suspend)
	return fctl.NewCommand(
		name+" <connectorinstance>",
		fctl.WithShortDescription(description),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorInstanceNames(factory)),
		fctl.WithConfirmFlag(),
		fctl.WithController[*SuspensionStore](controller),
	)
}

func (c *SuspensionController) GetStore() *SuspensionStore {
	return c.store
}

func (c *SuspensionController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}
	name := args[0]
	action := "suspend"
	if !c.suspend {
		action = "resume"
	}
	if !c.approve(cmd, "You are about to %s Connectivity connector instance %q", action, name) {
		return nil, fctl.ErrMissingApproval
	}
	_, err = client.PatchConnectorInstance(cmd.Context(), name, connectivityclient.ConnectorInstancePatch{
		"spec": map[string]any{"suspend": c.suspend},
	})
	if err != nil {
		return nil, err
	}
	c.store.Name = name
	c.store.Suspend = c.suspend
	return c, nil
}

func (c *SuspensionController) Render(cmd *cobra.Command, _ []string) error {
	request := "suspension"
	if !c.store.Suspend {
		request = "resumption"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Connector instance %q %s requested.\n", c.store.Name, request)
	return err
}
