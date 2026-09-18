package deployments

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

// Deployment status values polled by the wait loop. Mirrors the strings the
// deploy server emits — keep this list in sync with the server's status enum
// (terraform-hcp-proxy `internal/storage/models/deployment.go`). The server
// does not yet expose these as a typed enum in the OpenAPI spec; lift this
// block once it does so a rename produces a compile error instead of a
// silent polling deadlock.
const (
	statusApplied            = "applied"
	statusPlannedAndFinished = "planned_and_finished"
	statusErrored            = "errored"
)

type Create struct {
	*components.DeploymentResource
}

type CreateCtrl struct {
	store *Create
	wait  func(context.Context, time.Duration) error
}

var _ fctl.Controller[*Create] = (*CreateCtrl)(nil)

func newCreateStore() *Create {
	return &Create{}
}

func NewCreateCtrl() *CreateCtrl {
	return &CreateCtrl{
		store: newCreateStore(),
		wait:  waitForDeploymentPoll,
	}
}

func NewCreate() *cobra.Command {
	return newCreate(NewCreateCtrl())
}

func newCreate(ctrl *CreateCtrl) *cobra.Command {
	return fctl.NewCommand("create",
		fctl.WithShortDescription("Create a deployment (deploy an app)"),
		fctl.WithStringFlag("app-id", "", "App ID"),
		fctl.WithStringFlag("manifest-id", "", "Manifest ID to deploy"),
		fctl.WithIntFlag("manifest-version", 0, "Manifest version to deploy (required, >= 1)"),
		fctl.WithBoolFlag("wait", true, "Wait for the deployment to complete"),
		fctl.WithStringFlag("wait-timeout", "30m", "Max duration to wait for the deployment when --wait is set"),
		fctl.WithController(ctrl),
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
	manifestID := fctl.GetString(cmd, "manifest-id")
	if manifestID == "" {
		return nil, fmt.Errorf("manifest-id is required")
	}
	manifestVersion := fctl.GetInt(cmd, "manifest-version")
	if manifestVersion <= 0 {
		return nil, fmt.Errorf("manifest-version is required (>= 1)")
	}

	req := components.CreateDeploymentRequest{
		AppID:           appID,
		ManifestID:      manifestID,
		ManifestVersion: int64(manifestVersion),
	}

	cmd.SilenceUsage = true
	deployment, err := apiClient.CreateDeployment(cmd.Context(), req)
	if err != nil {
		return nil, err
	}
	c.store.DeploymentResource = &deployment.DeploymentResponse.Data

	if fctl.GetBool(cmd, "wait") {
		if err := c.waitDeploymentCompletion(cmd); err != nil {
			return nil, err
		}
	} else if c.store.Status == statusErrored {
		return nil, fmt.Errorf("deployment failed: %s", c.store.ID)
	}

	return c, nil
}

func waitForDeploymentPoll(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

func (c *CreateCtrl) waitDeploymentCompletion(cmd *cobra.Command) (err error) {

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
	spinner := pterm.DefaultSpinner.WithWriter(io.Discard)
	// Do not animate when the command output is captured or redirected.
	animate := false
	if file, ok := cmd.ErrOrStderr().(*os.File); ok {
		info, err := file.Stat()
		animate = err == nil && info.Mode()&os.ModeCharDevice != 0
	}

	if s := fctl.GetString(cmd, "output"); s == "plain" && animate {
		var err error
		spinner, err = spinner.WithWriter(cmd.ErrOrStderr()).Start("Waiting for deployment to complete...")
		if err != nil {
			return err
		}
		defer func() {
			if err := spinner.Stop(); err != nil {
				pterm.Error.Println(err)
			}
		}()
	}

	timeout := 30 * time.Minute
	if v := fctl.GetString(cmd, "wait-timeout"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid --wait-timeout: %w", err)
		}
		timeout = parsed
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	defer func() {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("timed out after %s waiting for deployment %s (last status: %s): %w",
				timeout, c.store.ID, c.store.Status, err)
		}
	}()

	waitFor := 0 * time.Second
	for {
		if err := c.wait(ctx, waitFor); err != nil {
			return err
		}
		// Backoff: 2s, 4s, 8s capped at 15s. A long-running terraform
		// apply doesn't need sub-second polling.
		if waitFor == 0 {
			waitFor = 2 * time.Second
		} else if waitFor < 15*time.Second {
			waitFor *= 2
			if waitFor > 15*time.Second {
				waitFor = 15 * time.Second
			}
		}
		r, err := apiClient.ReadDeployment(ctx, c.store.ID, nil)
		if err != nil {
			return err
		}
		c.store.DeploymentResource = &r.DeploymentResponse.Data

		spinner.UpdateText(fmt.Sprintf("Deployment status: %s", r.DeploymentResponse.Data.Status))
		switch r.DeploymentResponse.Data.Status {
		case statusApplied:
			spinner.UpdateText("Deployment completed successfully")
			return nil
		case statusPlannedAndFinished:
			spinner.UpdateText("Deployment completed successfully, no changes to apply")
			return nil
		case statusErrored:
			l, err := apiClient.ReadDeploymentLogs(ctx, c.store.ID)
			if err != nil {
				return fmt.Errorf("deployment failed: %s (could not read logs: %w)", c.store.ID, err)
			}

			var diagnostics strings.Builder
			for _, entry := range l.ReadLogsResponse.Data {
				if entry.Diagnostic != nil {
					fmt.Fprintf(&diagnostics, "\n%s: %s\n%s", entry.Diagnostic.Severity, entry.Diagnostic.Summary, entry.Diagnostic.Detail)
				} else if entry.Message != "" {
					fmt.Fprintf(&diagnostics, "\n%s", entry.Message)
				}
			}
			return fmt.Errorf("deployment failed: %s%s", c.store.ID, diagnostics.String())
		default:
			continue
		}
	}
}

func (c *CreateCtrl) Render(cmd *cobra.Command, args []string) error {
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
