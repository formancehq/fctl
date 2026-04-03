package shared

import (
	"fmt"
	"io"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	deployserverclient "github.com/formancehq/fctl/internal/deployserverclient/v3"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

// WaitRunCompletion polls for a run to reach a terminal state and returns the final run and any error logs.
func WaitRunCompletion(cmd *cobra.Command, apiClient *deployserverclient.DeployServer, runID string) (*components.Run, []components.Log, error) {
	spinner := &pterm.DefaultSpinner

	if s := fctl.GetString(cmd, "output"); s == "plain" {
		var err error
		spinner, err = spinner.Start("Waiting for deployment to complete...")
		if err != nil {
			return nil, nil, err
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
			return nil, nil, cmd.Context().Err()
		case <-time.After(waitFor):
			waitFor = 2 * time.Second
			r, err := apiClient.ReadRun(cmd.Context(), runID)
			if err != nil {
				return nil, nil, err
			}
			run := r.RunResponse.Data

			spinner.UpdateText(fmt.Sprintf("Deployment status: %s", run.Status))
			switch run.Status {
			case "applied":
				spinner.UpdateText("Deployment completed successfully")
				return &run, nil, nil
			case "planned_and_finished":
				spinner.UpdateText("Deployment completed successfully, no changes to apply")
				return &run, nil, nil
			case "errored":
				l, err := apiClient.ReadRunLogs(cmd.Context(), runID)
				if err != nil {
					return &run, nil, err
				}
				return &run, l.ReadLogsResponse.Data, nil
			default:
				continue
			}
		}
	}
}
