package connections

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type ShowStore struct {
	Connection connectivity.Connection `json:"connection"`
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
		fctl.WithValidArgsFunction(CompleteConnectionNames(factory)),
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
	response, err := client.GetConnection(cmd.Context(), operations.GetConnectionRequest{Name: url.PathEscape(args[0])}, nil)
	if err == nil && response != nil {
		err = connectivityinternal.ResponseError(response.StatusCode, response.Error)
	}
	var instance *connectivity.Connection
	if response != nil {
		instance = response.Connection
	}
	if err != nil {
		return nil, err
	}
	if instance == nil || instance.Name == "" {
		return nil, fmt.Errorf("show connectivity connection %q: empty response", args[0])
	}
	c.store.Connection = *instance
	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	instance := c.store.Connection
	out := cmd.OutOrStdout()
	if err := renderInstanceSection(out, "Information", pterm.TableData{{"Name", instance.Name}}); err != nil {
		return err
	}
	if err := renderInstanceSection(out, "Desired Configuration", pterm.TableData{
		{"Connector", stringValue(instance.ConnectionConfiguration.Connector)}, {"Version", stringValue(instance.ConnectionConfiguration.Version)},
		{"Channel", enumStringValue(instance.ConnectionConfiguration.Channel)}, {"Ledger", stringValue(instance.ConnectionConfiguration.Ledger)},
		{"Poll Interval", stringValue(instance.ConnectionConfiguration.PollInterval)}, {"Suspend", boolValue(instance.ConnectionConfiguration.Suspend)},
	}); err != nil {
		return err
	}
	if err := renderInstanceSection(out, "Last Recorded Status", pterm.TableData{
		{"Lifecycle", lifecycleValue(instance.ConnectionStatus)}, {"Applied Version", appliedChannelFloor(&instance)},
		{"Ingestion State", ingestionState(instance.ConnectionStatus)}, {"Progress Current", ingestionProgress(instance.ConnectionStatus)},
		{"Last Error (may be historical)", ingestionLastError(instance.ConnectionStatus)},
		{"Service Error", serviceError(instance.ConnectionStatus)}, {"Suspended by", instanceSuspensionSources(instance.ConnectionStatus)},
	}); err != nil {
		return err
	}
	fctl.Section.WithWriter(out).Println("Configuration")
	rows := configRows(&instance.ConnectionConfiguration)
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

func configRows(config *connectivity.ConnectionConfiguration) [][]string {
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
func instanceSuspensionSources(status *connectivity.ConnectionStatus) string {
	if status == nil {
		return ""
	}
	entries := make([]string, 0, len(status.SuspendedBy))
	for _, source := range status.SuspendedBy {
		entries = append(entries, strings.Join(nonEmptyStrings(string(source.Source), source.Name), "/"))
	}
	return strings.Join(entries, ", ")
}
func lifecycleValue(status *connectivity.ConnectionStatus) string {
	if status == nil {
		return ""
	}
	return string(status.Lifecycle)
}
func ingestionState(status *connectivity.ConnectionStatus) string {
	if status == nil || status.IngestionStatus == nil {
		return ""
	}
	return enumStringValue(status.IngestionStatus.State)
}
func ingestionProgress(status *connectivity.ConnectionStatus) string {
	if status == nil || status.IngestionStatus == nil || status.IngestionStatus.IngestionProgress == nil {
		return ""
	}
	return strconv.FormatInt(status.IngestionStatus.IngestionProgress.Current, 10)
}
func ingestionLastError(status *connectivity.ConnectionStatus) string {
	if status == nil || status.IngestionStatus == nil {
		return ""
	}
	return stringValue(status.IngestionStatus.LastError)
}
func serviceError(status *connectivity.ConnectionStatus) string {
	if status == nil || status.StatusError == nil {
		return ""
	}
	return string(status.StatusError.Code) + ": " + status.StatusError.Message
}
