package plugins

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ListStore struct {
	Plugins []connectivityclient.Plugin `json:"plugins"`
	Cursor  fctl.Cursor                 `json:"cursor"`
}

type ListController struct {
	factory connectivityinternal.ClientFactory
	store   *ListStore
}

var _ fctl.Controller[*ListStore] = (*ListController)(nil)

func NewListController(factory connectivityinternal.ClientFactory) *ListController {
	return &ListController{
		factory: factory,
		store:   &ListStore{Plugins: []connectivityclient.Plugin{}},
	}
}

func NewListCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewListController(factory)
	return fctl.NewCommand(
		"list",
		fctl.WithAliases("ls", "l"),
		fctl.WithShortDescription("List available Connectivity plugins"),
		fctl.WithArgs(cobra.ExactArgs(0)),
		fctl.WithValidArgsFunction(cobra.NoFileCompletions),
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

	response, err := client.ListPlugins(cmd.Context(), connectivityclient.ListOptions{
		Limit:    pageSize,
		Continue: cursor,
	})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("list connectivity plugins: empty response")
	}

	c.store.Plugins = response.Items
	c.store.Cursor = fctl.Cursor{PageSize: int64(pageSize)}
	if response.Continue != "" {
		c.store.Cursor.HasMore = true
		c.store.Cursor.Next = fctl.Ptr(response.Continue)
	}
	return c, nil
}

func (c *ListController) Render(cmd *cobra.Command, _ []string) error {
	rows := fctl.Map(c.store.Plugins, func(plugin connectivityclient.Plugin) []string {
		return []string{
			stringValue(plugin.Metadata.Name),
			stringValue(plugin.Spec.DefaultVersion),
			stringValue(plugin.Spec.Description),
			strings.Join(plugin.Spec.Capabilities, ", "),
			pluginPhase(plugin.Status),
		}
	})
	rows = fctl.Prepend(rows, []string{"Name", "Default Version", "Description", "Capabilities", "Phase"})
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

func pluginPhase(status *connectivityclient.PluginStatus) string {
	if status == nil {
		return ""
	}
	return stringValue(status.Phase)
}
