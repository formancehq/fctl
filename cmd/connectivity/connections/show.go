package connections

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	ConnectorInstance connectivityclient.ConnectorInstance `json:"connectorInstance"`
}

type ShowController struct {
	factory connectivityinternal.ClientFactory
	store   *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewShowController(factory connectivityinternal.ClientFactory) *ShowController {
	return &ShowController{factory: factory, store: &ShowStore{}}
}

func NewShowCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewShowController(factory)
	return fctl.NewCommand(
		"show <connection>",
		fctl.WithAliases("get", "g", "sh", "s"),
		fctl.WithShortDescription("Show a Connectivity connection"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorInstanceNames(factory)),
		fctl.WithController[*ShowStore](controller),
	)
}

func (c *ShowController) GetStore() *ShowStore {
	return c.store
}

func (c *ShowController) Run(cmd *cobra.Command, args []string) (fctl.Renderable, error) {
	if c.factory == nil {
		return nil, fmt.Errorf("connectivity client factory is required")
	}
	client, err := c.factory(cmd)
	if err != nil {
		return nil, err
	}
	instance, err := client.GetConnectorInstance(cmd.Context(), args[0])
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("show connectivity connection %q: empty response", args[0])
	}
	c.store.ConnectorInstance = *instance
	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	instance := c.store.ConnectorInstance
	out := cmd.OutOrStdout()
	if err := renderInstanceSection(out, "Information", pterm.TableData{{"Name", instance.Name}}); err != nil {
		return err
	}
	if err := renderInstanceSection(out, "Desired Configuration", pterm.TableData{
		{"Connector", instance.Configuration.Connector}, {"Version", stringValue(instance.Configuration.Version)},
		{"Channel", stringValue(instance.Configuration.Channel)}, {"Ledger", instance.Configuration.Ledger},
		{"Poll Interval", stringValue(instance.Configuration.PollInterval)}, {"Suspend", boolValue(instance.Configuration.Suspend)},
	}); err != nil {
		return err
	}
	if err := renderInstanceSection(out, "Last Recorded Status", pterm.TableData{
		{"Lifecycle", lifecycleValue(instance.Status)}, {"Applied Version", appliedChannelFloor(&instance)},
		{"Ingestion State", ingestionState(instance.Status)}, {"Progress Current", ingestionProgress(instance.Status)},
		{"Last Error (may be historical)", ingestionLastError(instance.Status)},
		{"Service Error", serviceError(instance.Status)}, {"Suspended by", instanceSuspensionSources(instance.Status)},
	}); err != nil {
		return err
	}
	fctl.Section.WithWriter(out).Println("Configuration")
	rows := configRows(instance.Configuration.Configuration())
	if len(rows) == 0 {
		_, err := fmt.Fprintln(out, "No configuration entries.")
		return err
	}
	rows = fctl.Prepend(rows, []string{"Key", "Kind", "Source"})
	return pterm.DefaultTable.WithHasHeader().WithWriter(out).WithData(rows).Render()
}

func renderInstanceSection(out io.Writer, title string, data pterm.TableData) error {
	fctl.Section.WithWriter(out).Println(title)
	return pterm.DefaultTable.WithWriter(out).WithData(data).Render()
}

func configRows(config *connectivityclient.ConnectorInstanceConfig) [][]string {
	if config == nil {
		return nil
	}
	rows := make([][]string, 0, len(config.Env)+len(config.Files))
	for key := range config.Env {
		rows = append(rows, []string{key, "environment", "literal"})
	}
	for key := range config.Files {
		rows = append(rows, []string{key, "file", "literal"})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	return rows
}

func boolValue(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}
func instanceSuspensionSources(status *connectivityclient.ConnectorInstanceStatus) string {
	if status == nil {
		return ""
	}
	entries := make([]string, 0, len(status.SuspendedBy))
	for _, source := range status.SuspendedBy {
		entries = append(entries, strings.Join(nonEmptyStrings(source.Source, source.Name), "/"))
	}
	return strings.Join(entries, ", ")
}
func lifecycleValue(status *connectivityclient.ConnectorInstanceStatus) string {
	if status == nil {
		return ""
	}
	return status.Lifecycle
}
func ingestionState(status *connectivityclient.ConnectorInstanceStatus) string {
	if status == nil || status.Ingestion == nil {
		return ""
	}
	return stringValue(status.Ingestion.State)
}
func ingestionProgress(status *connectivityclient.ConnectorInstanceStatus) string {
	if status == nil || status.Ingestion == nil || status.Ingestion.Progress == nil {
		return ""
	}
	return strconv.FormatInt(status.Ingestion.Progress.Current, 10)
}
func ingestionLastError(status *connectivityclient.ConnectorInstanceStatus) string {
	if status == nil || status.Ingestion == nil {
		return ""
	}
	return stringValue(status.Ingestion.LastError)
}
func serviceError(status *connectivityclient.ConnectorInstanceStatus) string {
	if status == nil || status.Error == nil {
		return ""
	}
	return status.Error.Code + ": " + status.Error.Message
}
