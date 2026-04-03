package apps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	"github.com/formancehq/fctl/v3/cmd/apps/printer"
	"github.com/formancehq/fctl/v3/cmd/apps/shared"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Deploy struct {
	*components.Run
	logs []components.Log
}

type DeployCtrl struct {
	store *Deploy
}

var _ fctl.Controller[*Deploy] = (*DeployCtrl)(nil)

func newDeployStore() *Deploy {
	return &Deploy{}
}

func NewDeployCtrl() *DeployCtrl {
	return &DeployCtrl{
		store: newDeployStore(),
	}
}

func NewDeploy() *cobra.Command {
	return fctl.NewCommand("deploy",
		fctl.WithShortDescription("Push manifest and deploy to a deployment"),
		fctl.WithStringFlag("name", "", "App name (required)"),
		fctl.WithStringFlag("path", "", "Path to the manifest file (required)"),
		fctl.WithStringFlag("deployment", "default", "Deployment name"),
		fctl.WithIntFlag("version", 0, "Manifest version to deploy (0 = latest)"),
		fctl.WithBoolFlag("wait", true, "Wait for the deployment to complete"),
		fctl.WithController(NewDeployCtrl()),
	)
}

func (c *DeployCtrl) GetStore() *Deploy {
	return c.store
}

func (c *DeployCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	appName := fctl.GetString(cmd, "name")
	if appName == "" {
		return nil, fmt.Errorf("--name is required")
	}
	path := fctl.GetString(cmd, "path")
	if path == "" {
		return nil, fmt.Errorf("--path is required")
	}
	deploymentName := fctl.GetString(cmd, "deployment")

	cmd.SilenceUsage = true

	// 1. Look up app by name
	appsRes, err := apiClient.ListApps(cmd.Context(), nil, nil)
	if err != nil {
		return nil, err
	}
	var appID string
	for _, app := range appsRes.ListAppsResponse.Data.Items {
		if app.Name == appName {
			appID = app.ID
			break
		}
	}
	if appID == "" {
		return nil, fmt.Errorf("app %q not found", appName)
	}

	// 2. Read and push manifest
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	_, err = apiClient.PushManifest(cmd.Context(), appID, data)
	if err != nil {
		return nil, fmt.Errorf("failed to push manifest: %w", err)
	}

	// 3. Deploy to deployment
	var versionPtr *int64
	if v := fctl.GetInt(cmd, "version"); v > 0 {
		v64 := int64(v)
		versionPtr = &v64
	}
	deployRes, err := apiClient.DeployToDeployment(cmd.Context(), appID, deploymentName, versionPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy: %w", err)
	}
	c.store.Run = &deployRes.RunResponse.Data

	// 4. Wait if requested
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
