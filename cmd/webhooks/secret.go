package webhooks

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type ChangeSecretStore struct {
	Secret string `json:"secret"`
	ID     string `json:"id"`
}

type ChangeSecretWebhookController struct {
	store *ChangeSecretStore
}

var _ fctl.Controller[*ChangeSecretStore] = (*ChangeSecretWebhookController)(nil)

func NewDefaultChangeSecretStore() *ChangeSecretStore {
	return &ChangeSecretStore{
		Secret: "",
		ID:     "",
	}
}
func NewChangeSecretWebhookController() *ChangeSecretWebhookController {
	return &ChangeSecretWebhookController{
		store: NewDefaultChangeSecretStore(),
	}
}
func (c *ChangeSecretWebhookController) GetStore() *ChangeSecretStore {
	return c.store
}
func (c *ChangeSecretWebhookController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if !fctl.CheckStackApprobation(cmd, "You are about to change a webhook secret") {
		return nil, fctl.ErrMissingApproval
	}
	secret := ""
	if len(args) > 1 {
		secret = args[1]
	}

	response, err := stackClient.Webhooks.V1.
		ChangeConfigSecret(cmd.Context(), operations.ChangeConfigSecretRequest{
			ConfigChangeSecret: &shared.ConfigChangeSecret{
				Secret: secret,
			},
			ID: args[0],
		})
	if err != nil {
		return nil, fmt.Errorf("changing secret: %w", err)
	}

	c.store.ID = response.ConfigResponse.Data.ID
	c.store.Secret = response.ConfigResponse.Data.Secret

	return c, nil
}

func (c *ChangeSecretWebhookController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln(
		"Config '%s' updated successfully with new secret", c.store.ID)
	return nil
}

func NewChangeSecretCommand() *cobra.Command {
	return fctl.NewCommand("change-secret <config-id> <secret>",
		fctl.WithShortDescription("Change the signing secret of a config. You can bring your own secret. If not passed or empty, a secret is automatically generated. The format is a string of bytes of size 24, base64 encoded. (larger size after encoding)"),
		fctl.WithConfirmFlag(),
		fctl.WithAliases("cs"),
		fctl.WithArgs(cobra.RangeArgs(1, 2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*ChangeSecretStore](NewChangeSecretWebhookController()),
	)
}
