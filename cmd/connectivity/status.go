package connectivity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	connectivitymodel "github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

type StatusController struct {
	store *connectivitymodel.Info
}

var _ fctl.Controller[*connectivitymodel.Info] = (*StatusController)(nil)

func NewStatusCommand() *cobra.Command {
	controller := &StatusController{store: &connectivitymodel.Info{}}
	return fctl.NewCommand("status",
		fctl.WithShortDescription("Show Connectivity service build information"),
		fctl.WithArgs(cobra.NoArgs),
		fctl.WithController[*connectivitymodel.Info](controller),
	)
}

func (c *StatusController) GetStore() *connectivitymodel.Info { return c.store }

func (c *StatusController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, stackClient.Connectivity.V1)
}

func (c *StatusController) run(cmd *cobra.Command, client *formance.FormanceV1) (fctl.Renderable, error) {
	response, err := client.ConnectivityInfo(cmd.Context())
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("get Connectivity build information: empty response")
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
