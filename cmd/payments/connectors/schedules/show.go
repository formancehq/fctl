package schedules

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v3/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	Schedule *shared.V3Schedule `json:"schedule"`
}

type ShowController struct {
	PaymentsVersion versions.Version

	store *ShowStore
}

func (c *ShowController) SetVersion(version versions.Version) {
	c.PaymentsVersion = version
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewShowStore() *ShowStore {
	return &ShowStore{}
}

func NewShowController() *ShowController {
	return &ShowController{
		store: NewShowStore(),
	}
}

func NewShowCommand() *cobra.Command {
	c := NewShowController()
	return fctl.NewCommand("get <connectorID> <scheduleID>",
		fctl.WithAliases("sh", "s"),
		fctl.WithArgs(cobra.ExactArgs(2)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithShortDescription("Get a connector schedule"),
		fctl.WithController[*ShowStore](c),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {

	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}

	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion.Major < versions.V3 {
		return nil, fmt.Errorf("connector schedules are only supported in >= v3.0.0")
	}

	connectorID, scheduleID := args[0], args[1]

	response, err := stackClient.Payments.V3.GetConnectorSchedule(cmd.Context(), operations.V3GetConnectorScheduleRequest{
		ConnectorID: connectorID,
		ScheduleID:  scheduleID,
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	if response.V3ConnectorScheduleResponse == nil {
		return nil, fmt.Errorf("unexpected response: %v", response)
	}

	c.store.Schedule = &response.V3ConnectorScheduleResponse.Data

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	s := c.store.Schedule

	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Information")
	tableData := pterm.TableData{
		{pterm.LightCyan("ID"), s.ID},
		{pterm.LightCyan("ConnectorID"), s.ConnectorID},
		{pterm.LightCyan("CreatedAt"), s.CreatedAt.Format(time.RFC3339)},
	}

	return pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render()
}
