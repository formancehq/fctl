package apps

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Init struct {
	App        components.App
	Deployment components.Deployment
}

type InitCtrl struct {
	store *Init
}

var _ fctl.Controller[*Init] = (*InitCtrl)(nil)

func newInitStore() *Init {
	return &Init{}
}

func NewInitCtrl() *InitCtrl {
	return &InitCtrl{
		store: newInitStore(),
	}
}

func NewInit() *cobra.Command {
	return fctl.NewCommand("init",
		fctl.WithShortDescription("Create an app and its default deployment"),
		fctl.WithStringFlag("name", "", "App name (required)"),
		fctl.WithStringFlag("stack-id", "", "Stack ID for the deployment (required)"),
		fctl.WithStringFlag("deployment", "default", "Deployment name"),
		fctl.WithController(NewInitCtrl()),
	)
}

func (c *InitCtrl) GetStore() *Init {
	return c.store
}

func (c *InitCtrl) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
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

	name := fctl.GetString(cmd, "name")
	if name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	stackID := fctl.GetString(cmd, "stack-id")
	if stackID == "" {
		return nil, fmt.Errorf("--stack-id is required")
	}
	deploymentName := fctl.GetString(cmd, "deployment")

	cmd.SilenceUsage = true

	// 1. Create the app
	appRes, err := apiClient.CreateApp(cmd.Context(), components.CreateAppRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}
	c.store.App = appRes.AppResponse.Data

	// 2. Create the deployment
	depRes, err := apiClient.CreateDeployment(cmd.Context(), c.store.App.ID, components.CreateDeploymentRequest{
		Name:    deploymentName,
		StackID: stackID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}
	c.store.Deployment = depRes.DeploymentResponse.Data

	return c, nil
}

func (c *InitCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Printfln("App initialized successfully")
	if err := pterm.
		DefaultTable.
		WithHasHeader().
		WithData([][]string{
			{"App ID", "App Name", "Deployment", "Stack ID"},
			{c.store.App.ID, c.store.App.Name, c.store.Deployment.Name, c.store.Deployment.StackID},
		}).
		WithWriter(cmd.OutOrStdout()).
		Render(); err != nil {
		return err
	}
	return nil
}
