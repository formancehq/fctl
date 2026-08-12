package connectorinstances

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

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
		"show <connectorinstance>",
		fctl.WithAliases("get", "g", "sh", "s"),
		fctl.WithShortDescription("Show a Connectivity connector instance"),
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
		return nil, fmt.Errorf("show connectivity connector instance %q: empty response", args[0])
	}
	c.store.ConnectorInstance = *instance
	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	instance := c.store.ConnectorInstance
	out := cmd.OutOrStdout()

	if err := renderInstanceSection(out, "Information", pterm.TableData{
		{pterm.LightCyan("Name"), stringValue(instance.Metadata.Name)},
		{pterm.LightCyan("Namespace"), stringValue(instance.Metadata.Namespace)},
		{pterm.LightCyan("UID"), stringValue(instance.Metadata.UID)},
		{pterm.LightCyan("Resource Version"), stringValue(instance.Metadata.ResourceVersion)},
		{pterm.LightCyan("Created At"), timeValue(instance.Metadata.CreationTimestamp)},
		{pterm.LightCyan("Labels"), mapValue(instance.Metadata.Labels)},
		{pterm.LightCyan("Annotations"), mapValue(instance.Metadata.Annotations)},
	}); err != nil {
		return err
	}
	if err := renderInstanceSection(out, "Desired Specification", pterm.TableData{
		{pterm.LightCyan("Connector"), instance.Spec.Connector},
		{pterm.LightCyan("Version"), stringValue(instance.Spec.Version)},
		{pterm.LightCyan("Channel"), stringValue(instance.Spec.Channel)},
		{pterm.LightCyan("Connectivity Reference"), stringValue(instance.Spec.ConnectivityRef)},
		{pterm.LightCyan("Ledger"), instance.Spec.Ledger},
		{pterm.LightCyan("Poll Interval"), stringValue(instance.Spec.PollInterval)},
	}); err != nil {
		return err
	}
	if err := renderInstanceSection(out, "Lifecycle", pterm.TableData{
		{pterm.LightCyan("Resolved Connector"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.ResolvedConnectorRef })},
		{pterm.LightCyan("Resolved Version"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.ResolvedVersion })},
		{pterm.LightCyan("Resolved Image"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.ResolvedImage })},
		{pterm.LightCyan("Resolved Digest"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.ResolvedDigest })},
		{pterm.LightCyan("Connector Address"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.ConnectorAddress })},
		{pterm.LightCyan("Phase"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.Phase })},
		{pterm.LightCyan("State"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.State })},
	}); err != nil {
		return err
	}
	if err := renderInstanceSection(out, "Ingestion Progress", pterm.TableData{
		{pterm.LightCyan("Current Sequence"), instanceStatusInt64(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *int64 { return status.CurrentSequence })},
		{pterm.LightCyan("Source Tip Sequence"), instanceStatusInt64(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *int64 { return status.SourceTipSequence })},
		{pterm.LightCyan("Last Error"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.LastError })},
		{pterm.LightCyan("Message"), instanceStatusValue(instance.Status, func(status *connectivityclient.ConnectorInstanceStatus) *string { return status.Message })},
	}); err != nil {
		return err
	}

	fctl.Section.WithWriter(out).Println("Configuration")
	rows := configRows(instance.Spec.Config)
	if len(rows) == 0 {
		_, err := fmt.Fprintln(out, "No configuration entries.")
		return err
	}
	rows = fctl.Prepend(rows, []string{"Key", "Kind", "Source", "Mode"})
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
	envKeys := make([]string, 0, len(config.Env))
	for key := range config.Env {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		value := config.Env[key]
		rows = append(rows, []string{key, "environment", configSource(value.Value, value.SecretRef, value.ConfigMapRef), ""})
	}
	files := append([]connectivityclient.FileMount(nil), config.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for _, file := range files {
		mode := ""
		if file.Mode != nil {
			mode = strconv.FormatInt(int64(*file.Mode), 10)
		}
		rows = append(rows, []string{file.Path, "file", configSource(file.Value, file.SecretRef, file.ConfigMapRef), mode})
	}
	return rows
}

func configSource(value *string, secretRef, configMapRef *connectivityclient.KeyRef) string {
	if secretRef != nil {
		return "secret:" + secretRef.Name + "/" + secretRef.Key
	}
	if configMapRef != nil {
		return "configmap:" + configMapRef.Name + "/" + configMapRef.Key
	}
	if value != nil {
		return "inline"
	}
	return ""
}

func timeValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func mapValue(value map[string]string) string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, key+"="+value[key])
	}
	return strings.Join(entries, ", ")
}
