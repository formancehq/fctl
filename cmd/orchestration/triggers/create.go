package triggers

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/pointer"

	fctl "github.com/formancehq/fctl/pkg"
)

type TriggersCreateStore struct {
	Trigger shared.Trigger `json:"trigger"`
}
type TriggersCreateController struct {
	store      *TriggersCreateStore
	nameFlag   string
	filterFlag string
	varsFlag   string
}

var _ fctl.Controller[*TriggersCreateStore] = (*TriggersCreateController)(nil)

func NewDefaultTriggersCreateStore() *TriggersCreateStore {
	return &TriggersCreateStore{}
}

func NewTriggersCreateController() *TriggersCreateController {
	return &TriggersCreateController{
		store:      NewDefaultTriggersCreateStore(),
		nameFlag:   "name",
		filterFlag: "filter",
		varsFlag:   "vars",
	}
}

func NewCreateCommand() *cobra.Command {
	ctrl := NewTriggersCreateController()
	return fctl.NewCommand("create <event> <workflow-id>",
		fctl.WithShortDescription("Create a trigger"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithController[*TriggersCreateStore](ctrl),
		fctl.WithStringFlag(ctrl.nameFlag, "", "Trigger's name"),
		fctl.WithStringFlag(ctrl.filterFlag, "", "Filter events"),
		fctl.WithStringSliceFlag(ctrl.varsFlag, []string{}, "Variables to pass to the workflow"),
	)
}

func (c *TriggersCreateController) GetStore() *TriggersCreateStore {
	return c.store
}

func (c *TriggersCreateController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	store := fctl.GetStackStore(cmd.Context())

	var (
		event    = args[0]
		name     = fctl.GetString(cmd, c.nameFlag)
		filter   = fctl.GetString(cmd, c.filterFlag)
		vars     = fctl.GetStringSlice(cmd, c.varsFlag)
		workflow = args[1]
	)

	data := &shared.V2TriggerData{
		Event:      event,
		Name:       &name,
		WorkflowID: workflow,
		Vars:       map[string]interface{}{},
	}
	if filter != "" {
		data.Filter = pointer.For(filter)
	}
	if len(vars) > 0 {
		for _, v := range vars {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) != 2 {
				return nil, errors.New("invalid 'vars' flag")
			}
			data.Vars[parts[0]] = parts[1]
		}
	}

	res, err := store.Client().Orchestration.V2.CreateTrigger(cmd.Context(), data)
	if err != nil {
		return nil, errors.Wrap(err, "reading trigger")
	}

	// Convert V2Trigger to Trigger
	v2Trigger := res.V2CreateTriggerResponse.Data
	c.store.Trigger = shared.Trigger{
		ID:         v2Trigger.ID,
		Name:       v2Trigger.Name,
		WorkflowID: v2Trigger.WorkflowID,
		Event:      v2Trigger.Event,
		Filter:     v2Trigger.Filter,
		Vars:       v2Trigger.Vars,
		CreatedAt:  v2Trigger.CreatedAt,
	}

	return c, nil
}

func (c *TriggersCreateController) Render(cmd *cobra.Command, args []string) error {
	pterm.Success.WithWriter(cmd.OutOrStdout()).Printfln("Trigger created with ID: %s", c.store.Trigger.ID)

	return nil
}
