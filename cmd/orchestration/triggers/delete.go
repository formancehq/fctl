package triggers

import (
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type TriggersDeleteStore struct {
	Success   bool   `json:"success"`
	TriggerID string `json:"triggerID"`
}
type TriggersDeleteController struct {
	store *TriggersDeleteStore
}

var _ fctl.Controller[*TriggersDeleteStore] = (*TriggersDeleteController)(nil)

func NewDefaultTriggersDeleteStore() *TriggersDeleteStore {
	return &TriggersDeleteStore{}
}

func NewTriggersDeleteController() *TriggersDeleteController {
	return &TriggersDeleteController{
		store: NewDefaultTriggersDeleteStore(),
	}
}

func NewDeleteCommand() *cobra.Command {
	return fctl.NewCommand("delete <trigger-id>",
		fctl.WithShortDescription("Delete a specific workflow trigger"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*TriggersDeleteStore](NewTriggersDeleteController()),
	)
}

func (c *TriggersDeleteController) GetStore() *TriggersDeleteStore {
	return c.store
}

func (c *TriggersDeleteController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	cfg, err := fctl.LoadConfig(cmd)
	if err != nil {
		return nil, err
	}

	profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, err
	}

	organizationID, stackID, err := fctl.ResolveStackID(cmd, *profile)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClient(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile, organizationID, stackID)
	if err != nil {
		return nil, err
	}
	_, err = stackClient.Orchestration.V1.DeleteTrigger(cmd.Context(), operations.DeleteTriggerRequest{
		TriggerID: args[0],
	})
	if err != nil {
		return nil, fmt.Errorf("deleting trigger: %w", err)
	}

	c.store.Success = true
	c.store.TriggerID = args[0]

	return c, nil
}

func (c *TriggersDeleteController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.Printfln("Trigger %s Deleted!", c.store.TriggerID)
	return nil
}
