package plugins

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

type ShowStore struct {
	Plugin connectivityclient.Plugin `json:"plugin"`
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
		"show <plugin>",
		fctl.WithAliases("get", "g", "sh", "s"),
		fctl.WithShortDescription("Show a Connectivity plugin"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompletePluginNames(factory)),
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
	plugin, err := client.GetPlugin(cmd.Context(), args[0])
	if err != nil {
		return nil, err
	}
	if plugin == nil {
		return nil, fmt.Errorf("show connectivity plugin %q: empty response", args[0])
	}
	c.store.Plugin = *plugin
	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	plugin := c.store.Plugin
	out := cmd.OutOrStdout()

	fctl.Section.WithWriter(out).Println("Information")
	information := pterm.TableData{
		{pterm.LightCyan("Name"), stringValue(plugin.Metadata.Name)},
		{pterm.LightCyan("Namespace"), stringValue(plugin.Metadata.Namespace)},
		{pterm.LightCyan("UID"), stringValue(plugin.Metadata.UID)},
		{pterm.LightCyan("Resource Version"), stringValue(plugin.Metadata.ResourceVersion)},
		{pterm.LightCyan("Created At"), timeValue(plugin.Metadata.CreationTimestamp)},
		{pterm.LightCyan("Labels"), mapValue(plugin.Metadata.Labels)},
		{pterm.LightCyan("Annotations"), mapValue(plugin.Metadata.Annotations)},
		{pterm.LightCyan("Image"), plugin.Spec.Image},
		{pterm.LightCyan("Version"), stringValue(plugin.Spec.Version)},
		{pterm.LightCyan("Default Version"), stringValue(plugin.Spec.DefaultVersion)},
		{pterm.LightCyan("Description"), stringValue(plugin.Spec.Description)},
		{pterm.LightCyan("Capabilities"), strings.Join(plugin.Spec.Capabilities, ", ")},
		{pterm.LightCyan("Phase"), pluginPhase(plugin.Status)},
		{pterm.LightCyan("Status Message"), pluginStatusMessage(plugin.Status)},
		{pterm.LightCyan("Documentation"), stringValue(plugin.Spec.DocsURL)},
	}
	if err := pterm.DefaultTable.WithWriter(out).WithData(information).Render(); err != nil {
		return err
	}

	if len(plugin.Spec.Versions) > 0 {
		fctl.Section.WithWriter(out).Println("Versions")
		versions := fctl.Map(plugin.Spec.Versions, func(version connectivityclient.VersionEntry) []string {
			return []string{version.Version, stringValue(version.Digest), stringValue(version.Image)}
		})
		versions = fctl.Prepend(versions, []string{"Version", "Digest", "Image"})
		if err := pterm.DefaultTable.WithHasHeader().WithWriter(out).WithData(versions).Render(); err != nil {
			return err
		}
	}

	fctl.Section.WithWriter(out).Println("Configuration Schema")
	schemaRows := summarizeSchema(plugin.Spec.ConfigSchema)
	if len(schemaRows) == 0 {
		_, err := fmt.Fprintln(out, "No configurable fields.")
		return err
	}
	schemaRows = fctl.Prepend(schemaRows, []string{"Key", "Source", "Requirement", "Format", "Description"})
	return pterm.DefaultTable.WithHasHeader().WithWriter(out).WithData(schemaRows).Render()
}

func summarizeSchema(schema map[string]any) [][]string {
	type field struct {
		key, source, requirement, format, description string
	}
	fields := make([]field, 0)
	sections := []struct {
		name   string
		source string
	}{
		{name: "env", source: "environment"},
		{name: "files", source: "file"},
	}

	for _, section := range sections {
		sectionSchema, ok := objectValue(schema[section.name])
		if !ok {
			continue
		}
		properties, _ := objectValue(sectionSchema["properties"])
		required := requiredKeys(sectionSchema["required"])
		for key, rawDefinition := range properties {
			definition, _ := objectValue(rawDefinition)
			format, _ := definition["format"].(string)
			if format == "" {
				format, _ = definition["type"].(string)
			}
			description, _ := definition["description"].(string)
			requirement := "optional"
			if required[key] {
				requirement = "required"
			}
			fields = append(fields, field{key, section.source, requirement, format, description})
		}
	}

	if len(fields) == 0 {
		properties, ok := objectValue(schema["properties"])
		if ok {
			required := requiredKeys(schema["required"])
			for key, rawDefinition := range properties {
				definition, _ := objectValue(rawDefinition)
				format, _ := definition["format"].(string)
				if format == "" {
					format, _ = definition["type"].(string)
				}
				description, _ := definition["description"].(string)
				requirement := "optional"
				if required[key] {
					requirement = "required"
				}
				fields = append(fields, field{key, "environment", requirement, format, description})
			}
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		if fields[i].source == fields[j].source {
			return fields[i].key < fields[j].key
		}
		return fields[i].source < fields[j].source
	})
	return fctl.Map(fields, func(value field) []string {
		return []string{value.key, value.source, value.requirement, value.format, value.description}
	})
}

func objectValue(value any) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	return object, ok
}

func requiredKeys(value any) map[string]bool {
	result := map[string]bool{}
	switch values := value.(type) {
	case []any:
		for _, value := range values {
			if key, ok := value.(string); ok {
				result[key] = true
			}
		}
	case []string:
		for _, key := range values {
			result[key] = true
		}
	}
	return result
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

func pluginStatusMessage(status *connectivityclient.PluginStatus) string {
	if status == nil {
		return ""
	}
	return stringValue(status.Message)
}
