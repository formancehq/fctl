package tasks

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/operations"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/cmd/payments/versions"
	fctl "github.com/formancehq/fctl/pkg"
)

type ShowStore struct {
	Task *shared.V3Task `json:"task"`
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
	return fctl.NewCommand("get <taskID>",
		fctl.WithShortDescription("Get task"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithAliases("sh", "s"),
		fctl.WithController[*ShowStore](c),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
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

	if err := versions.GetPaymentsVersion(cmd, args, c); err != nil {
		return nil, err
	}

	if c.PaymentsVersion < versions.V3 {
		return nil, fmt.Errorf("tasks are only supported in >= v3.0.0")
	}

	response, err := stackClient.Payments.V3.GetTask(cmd.Context(), operations.V3GetTaskRequest{
		TaskID: args[0],
	})
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	c.store.Task = &response.V3GetTaskResponse.Data

	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, args []string) error {
	var (
		errStr          string
		connectorID     string
		createdObjectID string
	)

	if c.store.Task.ConnectorID != nil {
		connectorID = *c.store.Task.ConnectorID
	}
	if c.store.Task.CreatedObjectID != nil {
		createdObjectID = *c.store.Task.CreatedObjectID
	}
	if c.store.Task.Error != nil {
		errStr = *c.store.Task.Error
	}

	fctl.Section.WithWriter(cmd.OutOrStdout()).Println("Information")
	tableData := pterm.TableData{}
	tableData = append(tableData, []string{pterm.LightCyan("ID"), c.store.Task.ID})
	tableData = append(tableData, []string{pterm.LightCyan("ConnectorID"), connectorID})
	tableData = append(tableData, []string{pterm.LightCyan("CreatedObjectID"), createdObjectID})
	tableData = append(tableData, []string{pterm.LightCyan("CreatedAt"), c.store.Task.CreatedAt.String()})
	tableData = append(tableData, []string{pterm.LightCyan("Error"), errStr})
	tableData = append(tableData, []string{pterm.LightCyan("Status"), string(c.store.Task.Status)})
	tableData = append(tableData, []string{pterm.LightCyan("UpdatedAt"), c.store.Task.UpdatedAt.String()})

	if err := pterm.DefaultTable.
		WithWriter(cmd.OutOrStdout()).
		WithData(tableData).
		Render(); err != nil {
		return err
	}

	return nil
}
