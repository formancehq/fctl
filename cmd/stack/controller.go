package stack

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/internal/membershipclient"
	"github.com/formancehq/fctl/internal/membershipclient/models/components"
	"github.com/formancehq/fctl/internal/membershipclient/models/operations"
)

func waitStackReady(cmd *cobra.Command, client *membershipclient.SDK, organizationId, stackId string) (*components.Stack, error) {
	waitTime := 2 * time.Second
	sum := 2 * time.Second

	// Hack to ignore first Status
	select {
	case <-cmd.Context().Done():
		return nil, cmd.Context().Err()
	case <-time.After(waitTime):
	}

	for {
		request := operations.GetStackRequest{
			OrganizationID: organizationId,
			StackID:        stackId,
		}

		stackRsp, err := client.GetStack(cmd.Context(), request)
		if err != nil {
			return nil, err
		}

		if stackRsp.GetHTTPMeta().Response.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("stack %s not found", stackId)
		}

		if stackRsp.CreateStackResponse == nil {
			return nil, fmt.Errorf("unexpected response: no data")
		}

		stackData := stackRsp.CreateStackResponse.GetData()

		if stackData == nil {
			return nil, fmt.Errorf("unexpected response: stack data is nil")
		}

		if stackData.GetStatus() == "READY" {
			return stackData, nil
		}

		if sum > 10*time.Minute {
			pterm.Warning.Printf("You can check fctl stack show %s --organization %s to see the status of the stack", stackId, organizationId)
			problem := fmt.Errorf("there might a problem with the stack scheduling, if the problem persists, please contact the support")

			err = internal.PrintStackInformation(cmd.OutOrStdout(), stackData, nil)
			if err != nil {
				return nil, problem
			}

			return nil, problem
		}

		sum += waitTime
		select {
		case <-time.After(waitTime):
		case <-cmd.Context().Done():
			return nil, cmd.Context().Err()
		}
	}
}
