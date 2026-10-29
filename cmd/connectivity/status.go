package connectivity

import (
	"fmt"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type StatusController struct {
	factory connectivityinternal.ClientFactory
	store   *connectivityclient.ServiceInfo
}

var _ fctl.Controller[*connectivityclient.ServiceInfo] = (*StatusController)(nil)

func NewStatusCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := &StatusController{factory: factory, store: &connectivityclient.ServiceInfo{}}
	return fctl.NewCommand("status",
		fctl.WithShortDescription("Show Connectivity service build information"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithController[*connectivityclient.ServiceInfo](controller),
	)
}

func (c *StatusController) GetStore() *connectivityclient.ServiceInfo { return c.store }

func (c *StatusController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("connectivity client is required")
	}
	info, err := client.GetInfo(cmd.Context())
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("get Connectivity build information: empty response")
	}
	c.store = info
	return c, nil
}

func (c *StatusController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\nCommit: %s\nBuild Date: %s\n", c.store.Version, c.store.Commit, c.store.BuildDate)
	return err
}
