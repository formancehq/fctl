package instances

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type InstancesSendEventStore struct {
	Success    bool   `json:"success"`
	InstanceID string `json:"instanceId"`
	Event      string `json:"event"`
}
type InstancesSendEventController struct {
	store *InstancesSendEventStore
}

var _ fctl.Controller[*InstancesSendEventStore] = (*InstancesSendEventController)(nil)

func NewDefaultInstancesSendEventStore() *InstancesSendEventStore {
	return &InstancesSendEventStore{
		Success:    false,
		InstanceID: "",
		Event:      "",
	}
}

func NewInstancesSendEventController() *InstancesSendEventController {
	return &InstancesSendEventController{
		store: NewDefaultInstancesSendEventStore(),
	}
}

func NewSendEventCommand() *cobra.Command {
	return fctl.NewCommand("send-event <instance-id> <event>",
		fctl.WithShortDescription("Send an event to an instance"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*InstancesSendEventStore](NewInstancesSendEventController()),
	)
}

func (c *InstancesSendEventController) GetStore() *InstancesSendEventStore {
	return c.store
}

func (c *InstancesSendEventController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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
	_, err = stackClient.Orchestration.V1.SendEvent(cmd.Context(), operations.SendEventRequest{
		RequestBody: &operations.SendEventRequestBody{
			Name: args[1],
		},
		InstanceID: args[0],
	})

	if err != nil {
		return nil, err
	}

	c.store.Success = true
	c.store.InstanceID = args[0]
	c.store.Event = args[1]

	return c, nil
}

func (c *InstancesSendEventController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Event '%s' sent", args[1])
	return nil
}
