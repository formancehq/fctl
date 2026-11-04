package connectivity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	connectivitymodel "github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type StatusController struct {
	factory connectivityinternal.ClientFactory
	store   *connectivitymodel.Info
}

var _ fctl.Controller[*connectivitymodel.Info] = (*StatusController)(nil)

func NewStatusCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := &StatusController{factory: factory, store: &connectivitymodel.Info{}}
	return fctl.NewCommand("status",
		fctl.WithShortDescription("Show Connectivity service build information"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithController[*connectivitymodel.Info](controller),
	)
}

func (c *StatusController) GetStore() *connectivitymodel.Info { return c.store }

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
	response, err := client.ConnectivityInfo(cmd.Context())
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("get Connectivity build information: empty response")
	}
	if err := connectivityinternal.ResponseError(response.StatusCode, response.Error); err != nil {
		return nil, err
	}
	if response.Info == nil {
		return nil, fmt.Errorf("get Connectivity build information: empty response")
	}
	if response.RawResponse == nil || response.RawResponse.Body == nil {
		return nil, fmt.Errorf("get Connectivity build information: empty raw response")
	}
	// The SDK retains the original body but its string fields cannot distinguish
	// missing/null values from valid empty build information.
	body, err := io.ReadAll(response.RawResponse.Body)
	_ = response.RawResponse.Body.Close()
	response.RawResponse.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("read Connectivity build information: %w", err)
	}
	var fields struct {
		Version   *string `json:"version"`
		Commit    *string `json:"commit"`
		BuildDate *string `json:"buildDate"`
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("decode Connectivity build information: %w", err)
	}
	if fields.Version == nil || fields.Commit == nil || fields.BuildDate == nil {
		return nil, fmt.Errorf("get Connectivity build information: version, commit and buildDate must be strings")
	}
	c.store = response.Info
	return c, nil
}

func (c *StatusController) Render(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\nCommit: %s\nBuild Date: %s\n", c.store.Version, c.store.Commit, c.store.BuildDate)
	return err
}
