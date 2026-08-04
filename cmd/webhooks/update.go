package webhooks

import (
	"fmt"
	"net/url"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v4/pkg/models/operations"
	webhooksmodels "github.com/formancehq/formance-sdk-go/v4/pkg/models/webhooks"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type UpdateWebhookStore struct {
	Success bool `json:"success"`
}

type UpdateWebhookController struct {
	store *UpdateWebhookStore
}

func NewUpdateWebhookController() *UpdateWebhookController {
	return &UpdateWebhookController{store: &UpdateWebhookStore{}}
}

func (c *UpdateWebhookController) GetStore() *UpdateWebhookStore { return c.store }

func (c *UpdateWebhookController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckStackApprobation(cmd, "You are about to update webhook config %s", args[0]) {
		return nil, fctl.ErrMissingApproval
	}
	if _, err := url.ParseRequestURI(args[1]); err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	secret := fctl.GetString(cmd, secretFlag)
	request := operations.UpdateConfigRequest{
		ID: args[0],
		ConfigUser: webhooksmodels.ConfigUser{
			Endpoint:   args[1],
			EventTypes: args[2:],
		},
	}
	if cmd.Flags().Changed(secretFlag) {
		request.ConfigUser.Secret = &secret
	}
	response, err := stackClient.Webhooks.V1.UpdateConfig(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("updating config: %w", err)
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("updating config: unexpected status code %d", response.StatusCode)
	}
	c.store.Success = true
	return c, nil
}

func (c *UpdateWebhookController) Render(cmd *cobra.Command, _ []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Config updated successfully")
	return nil
}

func NewUpdateCommand() *cobra.Command {
	return fctl.NewCommand("update <config-id> <endpoint> [<event-type>...]",
		fctl.WithShortDescription("Update a webhook config. At least one event type is required."),
		fctl.WithAliases("up"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.MinimumNArgs(3)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(secretFlag, "", "Set the webhook signing secret. If omitted, Webhooks generates a new one"),
		fctl.WithController[*UpdateWebhookStore](NewUpdateWebhookController()),
	)
}
