package webhooks

import (
	"fmt"
	"net/url"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	secretFlag = "secret"
)

type CreateWebhookController struct {
	store *CreateWebhookStore
}

type CreateWebhookStore struct {
	Webhook shared.WebhooksConfig `json:"webhook"`
}

var _ fctl.Controller[*CreateWebhookStore] = (*CreateWebhookController)(nil)

func NewDefaultCreateWebhookStore() *CreateWebhookStore {
	return &CreateWebhookStore{
		Webhook: shared.WebhooksConfig{},
	}
}

func NewCreateWebhookController() *CreateWebhookController {
	return &CreateWebhookController{
		store: NewDefaultCreateWebhookStore(),
	}
}
func (c *CreateWebhookController) GetStore() *CreateWebhookStore {
	return c.store
}

func (c *CreateWebhookController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to create a webhook") {
		return nil, fctl.ErrMissingApproval
	}

	if _, err := url.Parse(args[0]); err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	secret := fctl.GetString(cmd, secretFlag)

	response, err := stackClient.Webhooks.V1.InsertConfig(cmd.Context(), shared.ConfigUser{
		Endpoint:   args[0],
		EventTypes: args[1:],
		Secret:     &secret,
	})

	if err != nil {
		return nil, fmt.Errorf("creating config: %w", err)
	}

	c.store.Webhook = response.ConfigResponse.Data

	return c, nil
}

func (c *CreateWebhookController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Config created successfully")
	return nil
}

func NewCreateCommand() *cobra.Command {

	return fctl.NewCommand("create <endpoint> [<event-type>...]",
		fctl.WithShortDescription("Create a new config. At least one event type is required."),
		fctl.WithAliases("cr"),
		fctl.WithConfirmFlag(),
		fctl.WithArgs(cobra.MinimumNArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(secretFlag, "", "Bring your own webhooks signing secret. If not passed or empty, a secret is automatically generated. The format is a string of bytes of size 24, base64 encoded. (larger size after encoding)"),
		fctl.WithController[*CreateWebhookStore](NewCreateWebhookController()),
	)
}
