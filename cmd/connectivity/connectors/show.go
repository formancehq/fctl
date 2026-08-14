package connectors

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

const (
	showVersionsPageSize = int32(100)
	showVersionsMaxPages = 20
)

type ShowStore struct {
	Connector connectivityclient.Connector                 `json:"connector"`
	Versions  []connectivityclient.ConnectorVersionSummary `json:"versions"`
	Version   *connectivityclient.ConnectorVersion         `json:"version,omitempty"`
}

type ShowController struct {
	factory connectivityinternal.ClientFactory
	store   *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewShowController(factory connectivityinternal.ClientFactory) *ShowController {
	return &ShowController{
		factory: factory,
		store:   &ShowStore{Versions: []connectivityclient.ConnectorVersionSummary{}},
	}
}

func NewShowCommand(factory connectivityinternal.ClientFactory) *cobra.Command {
	controller := NewShowController(factory)
	return fctl.NewCommand(
		"show <connector>",
		fctl.WithAliases("get", "g", "sh", "s"),
		fctl.WithShortDescription("Show a Connectivity connector"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorNames(factory)),
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
	name := args[0]
	connector, err := client.GetConnector(cmd.Context(), name)
	if err != nil {
		return nil, err
	}
	if connector == nil {
		return nil, fmt.Errorf("show connectivity connector %q: empty response", name)
	}
	c.store.Connector = *connector

	versions, err := connectivityinternal.CollectPages(showVersionsPageSize, showVersionsMaxPages,
		func(options connectivityclient.ListOptions) ([]connectivityclient.ConnectorVersionSummary, bool, string, error) {
			page, err := client.ListConnectorVersions(cmd.Context(), name, options)
			if err != nil {
				return nil, false, "", err
			}
			if page == nil {
				return nil, false, "", fmt.Errorf("show connectivity connector %q: empty version list response", name)
			}
			return page.Items, page.HasMore, page.Next, nil
		})
	if err != nil {
		return nil, err
	}
	c.store.Versions = versions

	// The server resolves the `latest` alias to the newest Validated version;
	// published-but-unresolvable catalogues degrade to the no-version notice.
	if len(versions) > 0 {
		version, err := client.GetConnectorVersion(cmd.Context(), name, connectivityclient.VersionAliasLatest)
		var apiErr *connectivityclient.APIError
		switch {
		case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
		case err != nil:
			return nil, err
		case version == nil:
			return nil, fmt.Errorf("show connectivity connector %q: empty latest version response", name)
		default:
			c.store.Version = version
		}
	}
	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	connector := c.store.Connector
	out := cmd.OutOrStdout()

	fctl.Section.WithWriter(out).Println("Information")
	information := pterm.TableData{
		{pterm.LightCyan("Name"), stringValue(connector.Metadata.Name)},
		{pterm.LightCyan("Namespace"), stringValue(connector.Metadata.Namespace)},
		{pterm.LightCyan("UID"), stringValue(connector.Metadata.UID)},
		{pterm.LightCyan("Resource Version"), stringValue(connector.Metadata.ResourceVersion)},
		{pterm.LightCyan("Created At"), timeValue(connector.Metadata.CreationTimestamp)},
		{pterm.LightCyan("Labels"), mapValue(connector.Metadata.Labels)},
		{pterm.LightCyan("Annotations"), mapValue(connector.Metadata.Annotations)},
		{pterm.LightCyan("Display Name"), stringValue(connector.Spec.DisplayName)},
		{pterm.LightCyan("Description"), stringValue(connector.Spec.Description)},
		{pterm.LightCyan("Image URL"), stringValue(connector.Spec.ImageURL)},
		{pterm.LightCyan("Catalog"), stringValue(connector.Spec.Catalog)},
		{pterm.LightCyan("Tags"), strings.Join(connector.Spec.Tags, ", ")},
		{pterm.LightCyan("Tagline"), stringValue(connector.Spec.Tagline)},
		{pterm.LightCyan("Latest Version"), stringValue(connector.Spec.LatestVersion)},
		{pterm.LightCyan("Phase"), connectorPhase(connector.Status)},
		{pterm.LightCyan("Status Message"), connectorStatusMessage(connector.Status)},
	}
	if err := pterm.DefaultTable.WithWriter(out).WithData(information).Render(); err != nil {
		return err
	}

	if len(c.store.Versions) > 0 {
		fctl.Section.WithWriter(out).Println("Versions")
		versions := fctl.Map(c.store.Versions, func(version connectivityclient.ConnectorVersionSummary) []string {
			return []string{version.Version, stringValue(version.Digest), version.Image, timeValue(version.ReleaseDate)}
		})
		versions = fctl.Prepend(versions, []string{"Version", "Digest", "Image", "Released"})
		if err := pterm.DefaultTable.WithHasHeader().WithWriter(out).WithData(versions).Render(); err != nil {
			return err
		}
	}

	if c.store.Version == nil {
		fctl.Section.WithWriter(out).Println("Configuration Schema")
		_, err := fmt.Fprintln(out, "No published version.")
		return err
	}

	fctl.Section.WithWriter(out).Println("Configuration Schema (" + c.store.Version.Version + ")")
	schemaRows := summarizeSchema(c.store.Version.ConfigSchema)
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

func connectorStatusMessage(status *connectivityclient.ConnectorStatus) string {
	if status == nil {
		return ""
	}
	return stringValue(status.Message)
}
