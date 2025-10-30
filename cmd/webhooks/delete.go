package webhooks

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/sdkerrors"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type DeleteWebhookStore struct {
	ErrorResponse *sdkerrors.WebhooksErrorResponse `json:"error"`
	Success       bool                             `json:"success"`
}

type DeleteWebhookController struct {
	store *DeleteWebhookStore
}

var _ fctl.Controller[*DeleteWebhookStore] = (*DeleteWebhookController)(nil)

func NewDefaultDeleteWebhookStore() *DeleteWebhookStore {
	return &DeleteWebhookStore{
		Success: true,
	}
}

func NewDeleteWebhookController() *DeleteWebhookController {
	return &DeleteWebhookController{
		store: NewDefaultDeleteWebhookStore(),
	}
}

func (c *DeleteWebhookController) GetStore() *DeleteWebhookStore {
	return c.store
}

func (c *DeleteWebhookController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, err := fctl.ResolveOrganizationID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackID, err := fctl.ResolveStackID(cmd, *profile, organizationID)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), cfg.CurrentProfile, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to delete a webhook") {
		return nil, fctl.ErrMissingApproval
	}

	request := operations.DeleteConfigRequest{
		ID: args[0],
	}
	_, err = stackClient.Webhooks.V1.DeleteConfig(cmd.Context(), request)
	if err != nil {
		return nil, errors.Wrap(err, "deleting config")
	}

	c.store.Success = true

	return c, nil
}

func (c *DeleteWebhookController) Render(cmd *cobra.Command, args []string) error {
	if !c.store.Success {
		pterm.Warning.WithShowLineNumber(false).Printfln("Config %s not found", args[0])
		return nil
	}

	if c.store.ErrorResponse != nil {
		pterm.Warning.WithShowLineNumber(false).Printfln(c.store.ErrorResponse.ErrorMessage)
		return nil
	}

	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Config deleted successfully")

	return nil
}

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete <config-id>",
		fctl.WithShortDescription("Delete a config"),
		fctl.WithConfirmFlag(),
		fctl.WithAliases("del"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*DeleteWebhookStore](NewDeleteWebhookController()),
	)
}
