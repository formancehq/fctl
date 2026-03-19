package webhooks

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ActivateWebhookStore struct {
	Success bool `json:"success"`
}
type ActivateWebhookController struct {
	store *ActivateWebhookStore
}

func NewDefaultVersionStore() *ActivateWebhookStore {
	return &ActivateWebhookStore{
		Success: true,
	}
}
func NewActivateWebhookController() *ActivateWebhookController {
	return &ActivateWebhookController{
		store: NewDefaultVersionStore(),
	}
}
func (c *ActivateWebhookController) GetStore() *ActivateWebhookStore {
	return c.store
}

func (c *ActivateWebhookController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	if !fctl.CheckStackApprobation(cmd, "You are about to activate a webhook") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.ActivateConfigRequest{
		ID: args[0],
	}

	_, err = stackClient.Webhooks.V1.ActivateConfig(cmd.Context(), request)
	if err != nil {
		return nil, fmt.Errorf("activating config: %w", err)
	}

	return c, nil
}

func (*ActivateWebhookController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Config activated successfully")

	return nil
}

func NewActivateCommand() *cobra.Command {
	return fctl.NewCommand("activate <config-id>",
		fctl.WithShortDescription("Activate one config"),
		fctl.WithAliases("ac", "a"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*ActivateWebhookStore](NewActivateWebhookController()),
	)
}
