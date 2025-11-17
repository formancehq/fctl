package apps

import (
	"fmt"
	"os"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/cloud/apps/printer"
	"github.com/formancehq/fctl/internal/deployserverclient/models/components"
	fctl "github.com/formancehq/fctl/pkg"
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
		fctl.WithBoolFlag("wait", true, "Wait for the deployment to complete"),
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

	cmd.SilenceUsage = true
	deployment, err := store.Cli.DeployAppConfigurationRaw(cmd.Context(), id, data)
	if err != nil {
		return nil, err
	}
	c.store.Run = &deployment.RunResponse.Data
	return c, nil
}

func (c *DeployCtrl) waitRunCompletion(cmd *cobra.Command) error {
	store := fctl.GetDeployServerStore(cmd.Context())
	s, err := pterm.DefaultSpinner.Start("Waiting for deployment to complete...")
	if err != nil {
		return err
	}
	defer s.Stop()
	for {
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-time.After(2 * time.Second):
			r, err := store.Cli.ReadRun(cmd.Context(), c.store.ID)
			if err != nil {
				return err
			}

			s.InfoPrinter.Printf("\033[1A\033[K")
			s.InfoPrinter.Printfln("Deployment status: %s", r.RunResponse.Data.Status)
			switch r.RunResponse.Data.Status {
			case "applied":
				s.Success("Deployment completed successfully")
				return nil
			case "planned_and_finished":
				s.Success("Deployment completed successfully, no changes to apply")
				return nil
			case "errored": // TOFix: change it to show pro
				l, err := store.Cli.ReadRunLogs(cmd.Context(), c.store.ID)
				if err != nil {
					return err
				}

				if err := printer.RenderLogs(cmd.ErrOrStderr(), l.ReadLogsResponse.Data); err != nil {
					return err
				}
				return fmt.Errorf("deployment failed: %s", c.store.ID)
			default:
				continue
			}
		}
	}
}

func (c *DeployCtrl) Render(cmd *cobra.Command, args []string) error {
	pterm.Info.Println("App Deployment accepted", c.store.ID)
	wait := fctl.GetBool(cmd, "wait")
	if !wait {
		return nil
	}
	if err := c.waitRunCompletion(cmd); err != nil {
		return err
	}

	store := fctl.GetDeployServerStore(cmd.Context())
	id := fctl.GetString(cmd, "id")
	currentStateRes, err := store.Cli.ReadAppCurrentStateVersion(cmd.Context(), id)
	if err != nil {
		return err
	}
	if state := currentStateRes.GetReadStateResponse().Data.Stack; state != nil {
		cfg, err := fctl.GetConfig(cmd)
		membershipStore := fctl.GetMembershipStore(cmd.Context())
		organizationID, err := fctl.ResolveOrganizationID(cmd, cfg, membershipStore.Client())
		if err != nil {
			return nil
		}
		info, _, err := membershipStore.Client().GetServerInfo(cmd.Context()).Execute()
		if err != nil {
			return err
		}

		if info.ConsoleURL != nil {
			pterm.Success.Printfln("View stack in console: %s/%s/%s?region=%s", *info.ConsoleURL, organizationID, state["id"], state["region_id"])
		}
	}
	return nil
}
