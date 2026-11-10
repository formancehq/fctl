package connections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type ShowStore struct {
	Connection connectivity.Connection `json:"connection"`
	// Defaults lists the schema defaults of the applied version for keys the
	// connection does not set. Omitted when the schema cannot be read.
	Defaults map[string]string `json:"defaults,omitempty"`
}

type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewShowController() *ShowController {
	return &ShowController{store: &ShowStore{}}
}

func NewShowCommand() *cobra.Command {
	controller := NewShowController()
	return fctl.NewCommand(
		"show <connection>",
		fctl.WithAliases("get", "g", "sh", "s"),
		fctl.WithShortDescription("Show a connection's configuration and last recorded status"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectionNames()),
		fctl.WithController[*ShowStore](controller),
		fctl.WithExamples("fctl connectivity connections show stripeconnect-fpss"),
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
	client, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, client.Connectivity.V1)
}

func (c *ShowController) run(cmd *cobra.Command, args []string, client *sdk.FormanceV1) (fctl.Renderable, error) {
	response, err := client.GetConnection(cmd.Context(), operations.GetConnectionRequest{Name: url.PathEscape(args[0])}, nil)
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
	c.store.Defaults = schemaDefaults(cmd.Context(), client, instance)
	return c, nil
}

// schemaDefaults reads the applied version's schema, else the pinned one.
// It is best effort: show still renders the connection when the catalogue
// cannot be read, it only loses the default rows.
func schemaDefaults(ctx context.Context, client *sdk.FormanceV1, instance *connectivity.Connection) map[string]string {
	config := instance.ConnectionConfiguration
	connector := fctl.StringPointerToString(config.Connector)
	selector := fallback(appliedChannelFloor(instance), fctl.StringPointerToString(config.Version))
	if connector == "" || selector == "" {
		return nil
	}
	version, err := resolveConnectorVersion(ctx, client, connector, selector)
	if err != nil {
		return nil
	}
	fields, err := configuration.SchemaFields(version)
	if err != nil {
		return nil
	}
	defaults := make(map[string]string)
	for key, field := range fields {
		if !field.HasDefault {
			continue
		}
		if _, set := config.Env[key]; set {
			continue
		}
		if _, set := config.Files[key]; set {
			continue
		}
		defaults[key] = defaultText(field.Default)
	}
	if len(defaults) == 0 {
		return nil
	}
	return defaults
}

func defaultText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	instance := c.store.Connection
	config := instance.ConnectionConfiguration
	out := cmd.OutOrStdout()

	fctl.Section.WithWriter(out).Println("Connection")
	if err := fctl.RenderKeyValues(out, [][]string{
		{"Name", instance.Name},
		{"Connector", dash(fctl.StringPointerToString(config.Connector))},
		{"Ledger", dash(fctl.StringPointerToString(config.Ledger))},
		{"Target", targetDescription(&config)},
		{"Poll Interval", fallback(fctl.StringPointerToString(config.PollInterval), "service default")},
		{"Suspend", suspendValue(config.Suspend)},
	}); err != nil {
		return err
	}

	fctl.Section.WithWriter(out).Println("Last Recorded Status")
	lastError := "none recorded"
	if value := connectionLastError(instance.ConnectionStatus); value != "" {
		lastError = "(may be historical) " + value
	}
	ingestion := dash(ingestionState(instance.ConnectionStatus))
	if progress := ingestionProgress(instance.ConnectionStatus); progress != "" {
		ingestion += " (progress " + progress + ")"
	}
	if err := fctl.RenderKeyValues(out, [][]string{
		{"Lifecycle", lifecycleStyle(dash(lifecycleValue(instance.ConnectionStatus)))},
		{"Applied Version", dash(appliedChannelFloor(&instance))},
		{"Ingestion", ingestion},
		{"Suspended By", dash(instanceSuspensionSources(instance.ConnectionStatus))},
		{"Last Error", lastError},
	}); err != nil {
		return err
	}

	rows := configRows(&instance.ConnectionConfiguration, c.store.Defaults)
	if len(rows) == 0 {
		fctl.Section.WithWriter(out).Println("Configuration")
		_, err := fmt.Fprintln(out, "No configuration entries.")
		return err
	}
	values := "values"
	if len(rows) == 1 {
		values = "value"
	}
	fctl.Section.WithWriter(out).Println(fmt.Sprintf("Configuration (%d %s)", len(rows), values))
	columns := []fctl.TableColumn{{}, {}, {Fit: fctl.FitCut, Min: 12}}
	if err := fctl.RenderTable(out, []string{"Key", "Kind", "Value"}, columns, rows); err != nil {
		return err
	}
	return fctl.RenderText(out, pterm.Gray("Secret values (format: password) are never returned by the API."))
}

// configRows lists returned values; multi-line file contents are summarized
// by their first line, -o json carries them in full.
func configRows(config *connectivity.ConnectionConfiguration, defaults map[string]string) [][]string {
	if config == nil {
		return nil
	}
	rows := make([][]string, 0, len(config.Env)+len(config.Files)+len(defaults))
	for key, value := range defaults {
		rows = append(rows, []string{key, "default", value})
	}
	for key, value := range config.Env {
		rows = append(rows, []string{key, "env", value})
	}
	for key, value := range config.Files {
		lines := strings.Split(strings.TrimRight(value, "\n"), "\n")
		if len(lines) > 1 {
			value = fmt.Sprintf("%s … (%d lines)", lines[0], len(lines))
		}
		rows = append(rows, []string{key, "file", value})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	return rows
}

func targetDescription(config *connectivity.ConnectionConfiguration) string {
	if version := fctl.StringPointerToString(config.Version); version != "" {
		return "version " + version
	}
	if channel := enumStringValue(config.Channel); channel != "" {
		return "channel " + channel
	}
	return "-"
}

// suspendValue shows the desired state as the API returns it; the observed
// state is in the Last Recorded Status section.
func suspendValue(suspend *bool) string {
	if suspend == nil {
		return "-"
	}
	return strconv.FormatBool(*suspend)
}

func dash(value string) string {
	return fallback(value, "-")
}

func fallback(value, empty string) string {
	if value == "" {
		return empty
	}
	return value
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
func connectionLastError(status *connectivity.ConnectionStatus) string {
	if status == nil {
		return ""
	}
	return fctl.StringPointerToString(status.LastError)
}
