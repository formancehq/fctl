package connections

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
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
	return newSuspensionCommand(factory, "suspend", "Suspend a Connectivity connection", true)
}

func NewUnsuspendCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	return newSuspensionCommand(factory, "unsuspend", "Resume a Connectivity connection", false)
}

func newSuspensionCommand(factory connectivityinternal.ClientFactory, name, description string, suspend bool) *cobra.Command {
	controller := NewSuspensionController(factory, suspend)
	return fctl.NewCommand(
		name+" <connection>",
		fctl.WithShortDescription(description),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectionNames(factory)),
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
	if !c.approve(cmd, "You are about to %s Connectivity connection %q", action, name) {
		return nil, fctl.ErrMissingApproval
	}
	response, err := client.PatchConnection(cmd.Context(), operations.PatchConnectionRequest{Name: url.PathEscape(name), ConnectionPatch: connectivity.ConnectionPatch{Configuration: &connectivity.Configuration{Suspend: &c.suspend}}}, nil)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("suspend connectivity connection %q: empty response", name)
	}
	if err := connectivityinternal.ResponseError(response.StatusCode, response.Error); err != nil {
		return nil, err
	}
	if response.Connection == nil || response.Connection.Name == "" {
		return nil, fmt.Errorf("suspend connectivity connection %q: empty response", name)
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
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Connection %q %s requested.\n", c.store.Name, request)
	return err
}
