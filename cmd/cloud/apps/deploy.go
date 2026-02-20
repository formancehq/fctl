package apps

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"

	"github.com/formancehq/fctl/v3/cmd/cloud/apps/printer"
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
		return nil, fmt.Errorf("id is required")
	}
	path := fctl.GetString(cmd, "path")
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	cmd.SilenceUsage = true
	deployment, err := apiClient.DeployAppConfigurationRaw(cmd.Context(), id, data)
	if err != nil {
		return nil, err
	}
	c.store.Run = &deployment.RunResponse.Data

	if fctl.GetBool(cmd, "wait") {
		if err := c.waitRunCompletion(cmd); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *DeployCtrl) waitRunCompletion(cmd *cobra.Command) error {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return err
	}

	_, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return err
	}
	spinner := &pterm.DefaultSpinner

	if s := fctl.GetString(cmd, "output"); s == "plain" {
		var err error
		spinner, err = spinner.Start("Waiting for deployment to complete...")
		if err != nil {
			return err
		}
		defer func() {
			if err := spinner.Stop(); err != nil {
				pterm.Error.Println(err)
			}
		}()
	} else {
		spinner.SetWriter(io.Discard)
	}
	defer func() {
		_ = spinner.Stop()
	}()

	waitFor := 0 * time.Second
	for {
		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-time.After(waitFor):
			waitFor = 2 * time.Second
			r, err := apiClient.ReadRun(cmd.Context(), c.store.ID)
			if err != nil {
				return err
			}
			c.store.Run = &r.RunResponse.Data

			spinner.UpdateText(fmt.Sprintf("Deployment status: %s", r.RunResponse.Data.Status))
			switch r.RunResponse.Data.Status {
			case "applied":
				spinner.UpdateText("Deployment completed successfully")
				return nil
			case "planned_and_finished":
				spinner.UpdateText("Deployment completed successfully, no changes to apply")
				return nil
			case "errored":
				l, err := apiClient.ReadRunLogs(cmd.Context(), c.store.ID)
				if err != nil {
					return err
				}

				c.store.logs = l.ReadLogsResponse.Data

				return nil
			default:
				continue
			}
		}
	}
}

func (c *DeployCtrl) Render(cmd *cobra.Command, args []string) error {
	if c.store.Run.Status == "errored" {
		if len(c.store.logs) > 0 {
			if err := printer.RenderLogs(cmd.ErrOrStderr(), c.store.logs); err != nil {
				return err
			}
		}
		return fmt.Errorf("deployment failed: %s", c.store.ID)
	}

	pterm.Info.Println("App Deployment accepted", c.store.ID)
	wait := fctl.GetBool(cmd, "wait")
	if !wait {
		return nil
	}
	if err := c.waitRunCompletion(cmd); err != nil {
		return err
	}

	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return err
	}

	profile, profileName, err := fctl.LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return err
	}

	relyingParty, err := fctl.GetAuthRelyingParty(cmd.Context(), fctl.GetHttpClient(cmd), profile.MembershipURI)
	if err != nil {
		return err
	}

	organizationID, apiClient, err := fctl.NewAppDeployClientFromFlags(
		cmd,
		relyingParty,
		fctl.NewPTermDialog(),
		profileName,
		*profile,
	)
	if err != nil {
		return err
	}
	id := fctl.GetString(cmd, "id")
	currentStateRes, err := apiClient.ReadAppCurrentStateVersion(cmd.Context(), id)
	if err != nil {
		return err
	}
	if state := currentStateRes.GetReadStateResponse().Data.Stack; state != nil {

		apiClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
		if err != nil {
			return err
		}

		info, err := apiClient.GetServerInfo(cmd.Context())
		if err != nil {
			return err
		}

		if info.ServerInfo.ConsoleURL != nil {
			pterm.Success.Printfln("View stack in console: %s/%s/%s?region=%s", *info.ServerInfo.ConsoleURL, organizationID, state["id"], state["region_id"])
		}
	}
	return nil
}
