package apps

import (
	"fmt"
	"os"
	"time"

	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type Deploy struct {
	*components.Run
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

const (
	IdFlag   = "id"
	PathFlag = "path"
)

func NewDeploy() *cobra.Command {
	return fctl.NewCommand("deploy",
		fctl.WithShortDescription("Deploy apps"),
		fctl.WithStringFlag("id", "", "App ID"),
		fctl.WithStringFlag("path", "", "Path to the manifest file"),
		fctl.WithBoolFlag("wait", false, "Wait for the deployment to complete"),
		fctl.WithController(NewDeployCtrl()),
	)
}

func (c *DeployCtrl) GetStore() *Deploy {
	return c.store
}

func (c *DeployCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	path := fctl.GetString(cmd, "path")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	deployment, err := store.Cli.DeployAppConfigurationRaw(cmd.Context(), id, data)
	if err != nil {
		return nil, err
	}
	c.store.Run = &deployment.RunResponse.Data

	wait := fctl.GetBool(cmd, "wait")
	if !wait {
		return c, nil
	}

	pterm.Success.Println("App Deployement accepted: ", c.store.ID)
	s, err := pterm.DefaultSpinner.Start("Waiting for deployment to complete...")
	if err != nil {
		return nil, err
	}
	defer s.Stop()

	for {
		select {
		case <-cmd.Context().Done():
			return nil, cmd.Context().Err()
		case <-time.After(2 * time.Second):
			r, err := store.Cli.ReadRun(cmd.Context(), c.store.ID)
			if err != nil {
				return nil, err
			}
			switch r.RunResponse.Data.Status {
			case "applied":
				s.Success("Deployment completed successfully")
				return c, nil
			case "planned_and_finished":
				s.Success("Deployment completed successfully, no changes to apply")
				return c, nil
			case "errored":
				s.Fail("Deployment failed")
				// TOFix: change it to show pro
				l, err := store.Cli.ReadCurrentRunLogs(cmd.Context(), c.store.ID)
				if err != nil {
					return nil, err
				}

				data := [][]string{
					{"Timestamp", "Severity", "Summary", "Details"},
				}
				for _, entry := range l.ReadLogsResponse.Data {
					data = append(data, []string{
						entry.Timestamp.String(),
						entry.Diagnostic.Severity,
						entry.Diagnostic.Summary,
						entry.Diagnostic.Detail,
					})
				}
				return nil, fmt.Errorf("deployment failed: %s", c.store.ID)
			default:
				continue
			}
		}
	}
}

func (c *DeployCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Println("App Deployement accepted", c.store.ID)
	return nil
}
