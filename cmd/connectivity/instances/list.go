package instances

import (
	"fmt"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const pluginFlag = "plugin"

type ListStore struct {
	Instances []connectivityclient.Instance `json:"instances"`
	Cursor    fctl.Cursor                   `json:"cursor"`
}

type ListController struct {
	factory connectivityinternal.ClientFactory
	store   *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListController(factory connectivityinternal.ClientFactory) *ListController {
	return &ListController{
		factory: factory,
		store:   &ListStore{Instances: []connectivityclient.Instance{}},
	}
}

func NewListCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewListController(factory)
	return fctl.NewCommand(
		"list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List Connectivity instances"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
		fctl.WithStringFlag(pluginFlag, "", "Filter instances by plugin"),
		fctl.WithPageSizeFlag(),
		fctl.WithCursorFlag(),
		fctl.WithController[*ListStore](controller),
	)
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

	response, err := client.ListInstances(cmd.Context(), connectivityclient.ListOptions{
		Plugin:   fctl.GetString(cmd, pluginFlag),
		Limit:    pageSize,
		Continue: cursor,
	})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("list connectivity instances: empty response")
	}

	c.store.Instances = response.Items
	c.store.Cursor = fctl.Cursor{PageSize: int64(pageSize)}
	if response.Continue != "" {
		c.store.Cursor.HasMore = true
		c.store.Cursor.Next = fctl.Ptr(response.Continue)
	}
	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	rows := fctl.Map(c.store.Instances, func(instance connectivityclient.Instance) []string {
		return []string{
			stringValue(instance.Metadata.Name),
			instance.Spec.Plugin,
			stringValue(instance.Spec.Version),
			instance.Spec.Ledger,
			instanceStatusValue(instance.Status, func(status *connectivityclient.InstanceStatus) *string { return status.Phase }),
			instanceStatusValue(instance.Status, func(status *connectivityclient.InstanceStatus) *string { return status.State }),
			instanceStatusInt64(instance.Status, func(status *connectivityclient.InstanceStatus) *int64 { return status.CurrentSequence }),
			instanceStatusInt64(instance.Status, func(status *connectivityclient.InstanceStatus) *int64 { return status.SourceTipSequence }),
			instanceStatusValue(instance.Status, func(status *connectivityclient.InstanceStatus) *string { return status.LastError }),
		}
	})
	rows = fctl.Prepend(rows, []string{
		"Name", "Plugin", "Version", "Ledger", "Phase", "State", "Current Sequence", "Source Tip Sequence", "Last Error",
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

func instanceStatusValue(status *connectivityclient.InstanceStatus, field func(*connectivityclient.InstanceStatus) *string) string {
	if status == nil {
		return ""
	}
	return stringValue(field(status))
}

func instanceStatusInt64(status *connectivityclient.InstanceStatus, field func(*connectivityclient.InstanceStatus) *int64) string {
	if status == nil {
		return ""
	}
	return int64Value(field(status))
}
