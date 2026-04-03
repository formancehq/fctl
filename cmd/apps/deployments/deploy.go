package deployments

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	"github.com/formancehq/fctl/v3/cmd/apps/printer"
	"github.com/formancehq/fctl/v3/cmd/apps/shared"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type DeployStore struct {
	*components.Run
	logs []components.Log
}

type DeployCtrl struct {
	store *DeployStore
}

var _ fctl.Controller[*DeployStore] = (*DeployCtrl)(nil)

func newDeployStore() *DeployStore {
	return &DeployStore{}
}

func NewDeployCtrl() *DeployCtrl {
	return &DeployCtrl{
		store: newDeployStore(),
	}
}

func NewDeploy() *cobra.Command {
	return fctl.NewCommand("deploy",
		fctl.WithShortDescription("Deploy a manifest version to a deployment"),
		fctl.WithStringFlag("id", "", "App ID (required)"),
		fctl.WithStringFlag("name", "", "Deployment name (required)"),
		fctl.WithIntFlag("version", 0, "Manifest version to deploy (0 = latest)"),
		fctl.WithBoolFlag("wait", true, "Wait for the deployment to complete"),
		fctl.WithController(NewDeployCtrl()),
	)
}

func (c *DeployCtrl) GetStore() *DeployStore {
	return c.store
}

func (c *DeployCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return nil, err
	}

	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("--id is required")
	}
	name := fctl.GetString(cmd, "name")
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}

	var versionPtr *int64
	if v := fctl.GetInt(cmd, "version"); v > 0 {
		v64 := int64(v)
		versionPtr = &v64
	}

	cmd.SilenceUsage = true

	res, err := apiClient.DeployToDeployment(cmd.Context(), id, name, versionPtr)
	if err != nil {
		return nil, err
	}
	c.store.Run = &res.RunResponse.Data

	if fctl.GetBool(cmd, "wait") {
		run, logs, err := shared.WaitRunCompletion(cmd, apiClient, c.store.Run.ID)
		if err != nil {
			return nil, err
		}
		c.store.Run = run
		c.store.logs = logs
	}

	return c, nil
}

func (c *DeployCtrl) Render(cmd *cobra.Command, args []string) error {
	if c.store.Run.Status == "errored" {
		if len(c.store.logs) > 0 {
			if err := printer.RenderLogs(cmd.ErrOrStderr(), c.store.logs); err != nil {
				return err
			}
		}
		return fmt.Errorf("deployment failed: %s", c.store.Run.ID)
	}

	pterm.Success.Printfln("Deployment %s: %s", c.store.Run.Status, c.store.Run.ID)
	return nil
}
