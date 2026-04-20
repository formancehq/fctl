package deployments

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"

	"github.com/formancehq/fctl/v3/cmd/cloud/apps/printer"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type Create struct {
	*components.DeploymentResource
	logs []components.Log
}

type CreateCtrl struct {
	store *Create
}

var _ fctl.Controller[*Create] = (*CreateCtrl)(nil)

func newCreateStore() *Create {
	return &Create{}
}

func NewCreateCtrl() *CreateCtrl {
	return &CreateCtrl{
		store: newCreateStore(),
	}
}

func NewCreate() *cobra.Command {
	return fctl.NewCommand("create",
		fctl.WithShortDescription("Create a deployment (deploy an app)"),
		fctl.WithStringFlag("app-id", "", "App ID"),
		fctl.WithStringFlag("path", "", "Path to the manifest file"),
		fctl.WithBoolFlag("wait", true, "Wait for the deployment to complete"),
		fctl.WithController(NewCreateCtrl()),
	)
}

func (c *CreateCtrl) GetStore() *Create {
	return c.store
}

func (c *CreateCtrl) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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
	appID := fctl.GetString(cmd, "app-id")
	if appID == "" {
		return nil, fmt.Errorf("app-id is required")
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
	deployment, err := apiClient.CreateDeploymentRaw(cmd.Context(), data, &appID)
	if err != nil {
		return nil, err
	}
	c.store.DeploymentResource = &deployment.DeploymentResponse.Data

	if fctl.GetBool(cmd, "wait") {
		if err := c.waitDeploymentCompletion(cmd); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *CreateCtrl) waitDeploymentCompletion(cmd *cobra.Command) error {

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
			r, err := apiClient.ReadDeployment(cmd.Context(), c.store.ID, nil)
			if err != nil {
				return err
			}
			c.store.DeploymentResource = &r.DeploymentResponse.Data

			spinner.UpdateText(fmt.Sprintf("Deployment status: %s", r.DeploymentResponse.Data.RunStatus))
			switch r.DeploymentResponse.Data.RunStatus {
			case "applied":
				spinner.UpdateText("Deployment completed successfully")
				return nil
			case "planned_and_finished":
				spinner.UpdateText("Deployment completed successfully, no changes to apply")
				return nil
			case "errored":
				l, err := apiClient.ReadDeploymentLogs(cmd.Context(), c.store.ID)
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

func (c *CreateCtrl) Render(cmd *cobra.Command, args []string) error {
	if c.store.DeploymentResource.RunStatus == "errored" {
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

	appID := fctl.GetString(cmd, "app-id")
	appResp, err := apiClient.ReadApp(cmd.Context(), appID, []operations.ReadAppInclude{operations.ReadAppIncludeState})
	if err != nil {
		return err
	}

	if state := appResp.AppResponse.Data.State; state != nil && state.Stack != nil {
		membershipClient, err := fctl.NewMembershipClientForOrganization(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID)
		if err != nil {
			return err
		}

		info, err := membershipClient.GetServerInfo(cmd.Context())
		if err != nil {
			return err
		}

		if info.ServerInfo.ConsoleURL != nil {
			pterm.Success.Printfln("View stack in console: %s/%s/%s?region=%s", *info.ServerInfo.ConsoleURL, organizationID, state.Stack["id"], state.Stack["region_id"])
		}
	}
	return nil
}
