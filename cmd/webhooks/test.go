package webhooks

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	webhooksmodels "github.com/formancehq/formance-sdk-go/v4/pkg/models/webhooks"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type TestWebhookStore struct {
	Attempt webhooksmodels.Attempt `json:"attempt"`
}

type TestWebhookController struct {
	store *TestWebhookStore
}

func NewTestWebhookController() *TestWebhookController {
	return &TestWebhookController{store: &TestWebhookStore{}}
}

func (c *TestWebhookController) GetStore() *TestWebhookStore { return c.store }

func (c *TestWebhookController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckStackApprobation(cmd, "You are about to send a test event to webhook config %s", args[0]) {
		return nil, fctl.ErrMissingApproval
	}
	response, err := stackClient.Webhooks.V1.TestConfig(cmd.Context(), operations.TestConfigRequest{ID: args[0]})
	if err != nil {
		return nil, fmt.Errorf("testing config: %w", err)
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("testing config: unexpected status code %d", response.StatusCode)
	}
	if response.AttemptResponse == nil {
		return nil, fmt.Errorf("testing config: empty response")
	}
	c.store.Attempt = response.AttemptResponse.Attempt
	return c, nil
}

func (c *TestWebhookController) Render(cmd *cobra.Command, _ []string) error {
	attempt := c.store.Attempt
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Test event sent")
	return pterm.DefaultTable.WithWriter(cmd.OutOrStdout()).WithData(pterm.TableData{
		{"Attempt ID", attempt.ID},
		{"Webhook ID", attempt.WebhookID},
		{"Status", attempt.Status},
		{"Status code", strconv.FormatInt(attempt.StatusCode, 10)},
		{"Retry attempt", strconv.FormatInt(attempt.RetryAttempt, 10)},
		{"Next retry after", optionalTime(attempt.NextRetryAfter)},
		{"Created at", attempt.CreatedAt.Format(time.RFC3339)},
		{"Updated at", attempt.UpdatedAt.Format(time.RFC3339)},
	}).Render()
}

func NewTestCommand() *cobra.Command {
	return fctl.NewCommand("test <config-id>",
		fctl.WithShortDescription("Send a test event to a webhook config"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*TestWebhookStore](NewTestWebhookController()),
	)
}
