package connectorinstances

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connectors"
	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const connectorFlag = "connector"

type ListStore struct {
	ConnectorInstances []connectivityclient.ConnectorInstance `json:"connectorInstances"`
	Cursor             fctl.Cursor                            `json:"cursor"`
}

type ListController struct {
	factory connectivityinternal.ClientFactory
	store   *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListController(factory connectivityinternal.ClientFactory) *ListController {
	return &ListController{
		factory: factory,
		store:   &ListStore{ConnectorInstances: []connectivityclient.ConnectorInstance{}},
	}
}

func NewListCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewListController(factory)
	command := fctl.NewCommand(
		"list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List Connectivity connector instances"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(connectorFlag, "", "Filter connector instances by connector"),
		connectivityinternal.WithListQueryFlags(),
		fctl.WithPageSizeFlag(),
		fctl.WithCursorFlag(),
		fctl.WithController[*ListStore](controller),
	)
	if err := command.RegisterFlagCompletionFunc(connectorFlag, connectors.CompleteConnectorNames(factory)); err != nil {
		panic(err)
	}
	if err := command.RegisterFlagCompletionFunc(
		connectivityinternal.FilterFlag,
		connectivityinternal.CompleteFilterExpressions(factory, connectivityclient.ResourceConnectorInstances),
	); err != nil {
		panic(err)
	}
	return command
}

func (c *ListController) GetStore() *ListStore {
	return c.store
}

func (c *ListController) Run(cmd *cobra.Command, _ []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}
	pageSize, err := fctl.GetPageSize(cmd)
	if err != nil {
		return nil, err
	}
	cursor, err := fctl.GetCursor(cmd)
	if err != nil {
		return nil, err
	}
	filters, err := cmd.Flags().GetStringArray(connectivityinternal.FilterFlag)
	if err != nil {
		return nil, err
	}
	// --connector is sugar for one more filter clause, so it composes with
	// --filter and conflicts with --query exactly like the other filters.
	if connector := fctl.GetString(cmd, connectorFlag); connector != "" {
		filters = append(filters, connectorFlag+"="+connector)
	}
	query, err := connectivityinternal.BuildListQuery(fctl.GetString(cmd, connectivityinternal.QueryFlag), filters)
	if err != nil {
		return nil, err
	}

	response, err := client.ListConnectorInstances(cmd.Context(), connectivityclient.ListOptions{
		Query:    query,
		PageSize: pageSize,
		Cursor:   cursor,
	})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("list connectivity connector instances: empty response")
	}

	c.store.ConnectorInstances = response.Items
	c.store.Cursor = fctl.Cursor{PageSize: int64(response.PageSize), HasMore: response.HasMore}
	if response.Next != "" {
		c.store.Cursor.Next = fctl.Ptr(response.Next)
	}
	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	rows := fctl.Map(c.store.ConnectorInstances, func(instance connectivityclient.ConnectorInstance) []string {
		return []string{
			stringValue(instance.Metadata.Name),
			instance.Spec.Connector,
			stringValue(instance.Spec.Version),
			stringValue(instance.Spec.Channel),
			instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.ResolvedVersion }),
			instance.Spec.Ledger,
			instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.Phase }),
			instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.State }),
			instanceStatusInt64(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *int64 { return status.CurrentSequence }),
			instanceStatusInt64(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *int64 { return status.SourceTipSequence }),
			instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.LastError }),
		}
	})
	rows = fctl.Prepend(rows, []string{
		"Name", "Connector", "Version", "Channel", "Resolved Version", "Ledger", "Phase", "State", "Current Sequence", "Source Tip Sequence", "Last Error",
	})
	if err := pterm.DefaultTable.
		WithHasHeader().
		WithWriter(cmd.OutOrStdout()).
		WithData(rows).
		Render(); err != nil {
		return err
	}
	return fctl.RenderCursor(cmd.OutOrStdout(), c.store.Cursor)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int64Value(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func instanceStatusValue(status *connectivityclient.ConnectorInstanceStatus, field func(*connectivityclient.ConnectorInstanceStatus) *string) string {
	if status == nil {
		return ""
	}
	return stringValue(field(status))
}

func instanceStatusInt64(status *connectivityclient.ConnectorInstanceStatus, field func(*connectivityclient.ConnectorInstanceStatus) *int64) string {
	if status == nil {
		return ""
	}
	return int64Value(field(status))
}
