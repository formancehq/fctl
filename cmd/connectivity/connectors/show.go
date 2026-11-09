package connectors

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const (
	showVersionsPageSize = int32(100)
	showVersionsMaxPages = 20
)

type ShowStore struct {
	Connector connectivity.Connector                 `json:"connector"`
	Versions  []connectivity.ConnectorVersionSummary `json:"versions"`
	Version   *connectivity.ConnectorVersion         `json:"version,omitempty"`
}

type ShowController struct {
	store *ShowStore
}

var _ fctl.Controller[*ShowStore] = (*ShowController)(nil)

func NewShowController() *ShowController {
	return &ShowController{
		store: &ShowStore{Versions: []connectivity.ConnectorVersionSummary{}},
	}
}

func NewShowCommand() *cobra.Command {
	controller := NewShowController()
	return fctl.NewCommand(
		"show <connector>",
		fctl.WithAliases("get", "g", "sh", "s"),
		fctl.WithShortDescription("Show a connector and its configuration fields"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorNames()),
		fctl.WithController[*ShowStore](controller),
		fctl.WithExamples(
			"fctl connectivity connectors show ee-stripeconnect",
			"fctl connectivity connectors show ee-stripeconnect -o json \\",
			"    | jq '.data.version.configSchema'",
		),
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
	stackClient, err := fctl.NewStackClientFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return c.run(cmd, args, stackClient.Connectivity.V1)
}

func (c *ShowController) run(cmd *cobra.Command, args []string, client *formance.FormanceV1) (fctl.Renderable, error) {
	name := args[0]
	response, err := client.GetConnector(cmd.Context(), operations.GetConnectorRequest{Name: url.PathEscape(name)}, nil)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Connector == nil {
		return nil, fmt.Errorf("show connectivity connector %q: empty response", name)
	}
	c.store.Connector = *response.Connector

	versions, err := connectivityinternal.CollectPages(showVersionsPageSize, showVersionsMaxPages,
		func(options connectivityinternal.ListOptions) ([]connectivity.ConnectorVersionSummary, bool, string, error) {
			response, err := client.ListConnectorVersions(cmd.Context(), operations.ListConnectorVersionsRequest{
				Name:     url.PathEscape(name),
				PageSize: fctl.Ptr(int(options.PageSize)),
				Cursor:   connectivityinternal.OptionalString(options.Cursor),
				Query:    connectivityinternal.OptionalString(options.Query),
			}, nil)
			if err != nil {
				return nil, false, "", err
			}
			if response == nil || response.ConnectorVersionCursor == nil {
				return nil, false, "", fmt.Errorf("show connectivity connector %q: empty version list response", name)
			}
			page := response.ConnectorVersionCursor.Cursor
			if page.Data == nil {
				return nil, false, "", fmt.Errorf("show connectivity connector %q: version cursor.data must be an array", name)
			}
			return page.Data, page.HasMore, fctl.StringPointerToString(page.Next), nil
		})
	if err != nil {
		return nil, err
	}
	c.store.Versions = versions

	// Fetch the exact release projected by the catalogue.
	if version := fctl.StringPointerToString(c.store.Connector.LatestVersion); version != "" {
		response, err := client.GetConnectorVersion(cmd.Context(), operations.GetConnectorVersionRequest{Name: url.PathEscape(name), Version: url.PathEscape(version)}, nil)
		if err != nil {
			return nil, err
		}
		if response == nil || response.ConnectorVersion == nil {
			return nil, fmt.Errorf("show connectivity connector %q: empty latest version response", name)
		}
		c.store.Version = response.ConnectorVersion
	}
	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	connector := c.store.Connector
	out := cmd.OutOrStdout()

	fctl.Section.WithWriter(out).Println("Information")
	if err := fctl.RenderKeyValues(out, [][]string{
		{"Name", connector.Name},
		{"Display Name", orDash(fctl.StringPointerToString(connector.DisplayName))},
		{"Tagline", orDash(fctl.StringPointerToString(connector.Tagline))},
		{"Description", orDash(fctl.StringPointerToString(connector.Description))},
		{"Tags", orDash(strings.Join(connector.Tags, ", "))},
		{"Catalog", orDash(fctl.StringPointerToString(connector.Catalog))},
		{"Created At", orDash(timeValue(connector.CreatedAt))},
		{"Latest Version", orDash(latestVersionValue(fctl.StringPointerToString(connector.LatestVersion), c.store.Version))},
		{"Available Versions", strconv.Itoa(len(c.store.Versions))},
	}); err != nil {
		return err
	}

	if c.store.Version == nil {
		fctl.Section.WithWriter(out).Println("Configuration")
		message := "No published version."
		if len(c.store.Versions) > 0 {
			message = "No latest version schema available."
		}
		_, err := fmt.Fprintln(out, message)
		return err
	}

	fields := schemaFields(c.store.Version.ConfigSchema)
	if len(fields) == 0 {
		fctl.Section.WithWriter(out).Println("Configuration (" + c.store.Version.Version + ")")
		_, err := fmt.Fprintln(out, "No configurable fields.")
		return err
	}
	required := 0
	for _, field := range fields {
		if field.required {
			required++
		}
	}
	fctl.Section.WithWriter(out).Println(fmt.Sprintf("Configuration (%s) · %d fields, %d required", c.store.Version.Version, len(fields), required))
	columns := []fctl.TableColumn{{}, {}, {Fit: fctl.FitCut, Min: 6}, {Fit: fctl.FitWrap, Min: 20, MaxLines: 3}}
	if err := fctl.RenderGroupedTable(out, []string{"Key", "Req", "Type", "Description"}, columns, schemaGroups(fields)); err != nil {
		return err
	}

	rules := schemaRules(c.store.Version.ConfigSchema)
	if len(rules) == 0 {
		return nil
	}
	fctl.Section.WithWriter(out).Println("Rules")
	for _, rule := range rules {
		if err := fctl.RenderBullet(out, rule); err != nil {
			return err
		}
	}
	return nil
}

type schemaField struct {
	key, source, kind, description, group string
	required, advanced                    bool
	order                                 int
}

// schemaFields lists the env and file properties of a configSchema in the
// order the producer chose in presentation.groups. Properties outside every
// group follow, sorted by source and key.
func schemaFields(schema map[string]any) []schemaField {
	type placement struct {
		title    string
		advanced bool
		order    int
	}
	placements := map[string]placement{}
	presentation, _ := objectValue(schema["presentation"])
	groups, _ := presentation["groups"].([]any)
	order := 0
	for _, raw := range groups {
		group, _ := objectValue(raw)
		title, _ := group["title"].(string)
		advanced, _ := group["advanced"].(bool)
		properties, _ := group["properties"].([]any)
		for _, reference := range properties {
			if name, ok := reference.(string); ok {
				placements[name] = placement{title: title, advanced: advanced, order: order}
				order++
			}
		}
	}

	fields := make([]schemaField, 0)
	sections := []struct{ name, source, reference string }{
		{name: "env", source: "environment", reference: "env."},
		{name: "files", source: "file", reference: "files"},
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
			kind, _ := definition["format"].(string)
			if kind == "" {
				kind, _ = definition["type"].(string)
			}
			if kind == "password" || definition["writeOnly"] == true {
				kind = "secret"
			}
			description, _ := definition["description"].(string)
			description = fctl.FirstSentence(description)
			if value, ok := definition["default"]; ok {
				description = strings.TrimSpace(description + " Default: " + defaultValue(value) + ".")
			}
			place, grouped := placements[section.reference+key]
			if !grouped {
				place = placement{order: math.MaxInt}
			}
			fields = append(fields, schemaField{key: key, source: section.source, kind: kind, description: description, group: place.title, required: required[key], advanced: place.advanced, order: place.order})
		}
	}
	sort.SliceStable(fields, func(i, j int) bool {
		if fields[i].order != fields[j].order {
			return fields[i].order < fields[j].order
		}
		if fields[i].source != fields[j].source {
			return fields[i].source < fields[j].source
		}
		return fields[i].key < fields[j].key
	})
	return fields
}

func schemaGroups(fields []schemaField) []fctl.TableGroup {
	groups := make([]fctl.TableGroup, 0)
	for i, field := range fields {
		if i == 0 || field.group != fields[i-1].group {
			title := ""
			switch {
			case field.group != "" && field.advanced:
				title = "— " + field.group + " (advanced)"
			case field.group != "":
				title = "— " + field.group
			case i > 0:
				title = "— Other"
			}
			groups = append(groups, fctl.TableGroup{Title: title})
		}
		required := ""
		if field.required {
			required = "yes"
		}
		group := &groups[len(groups)-1]
		group.Rows = append(group.Rows, []string{field.key, required, field.kind, field.description})
	}
	return groups
}

// schemaRules returns the distinct descriptions of the schema requirements.
func schemaRules(schema map[string]any) []string {
	requirements, _ := schema["requirements"].([]any)
	seen := map[string]bool{}
	rules := make([]string, 0)
	for _, raw := range requirements {
		requirement, _ := objectValue(raw)
		description, _ := requirement["description"].(string)
		if description != "" && !seen[description] {
			seen[description] = true
			rules = append(rules, description)
		}
	}
	return rules
}

func defaultValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func latestVersionValue(latest string, version *connectivity.ConnectorVersion) string {
	if latest == "" || version == nil || version.Version != latest {
		return latest
	}
	details := make([]string, 0, 2)
	if version.ReleaseDate != nil {
		details = append(details, "released "+version.ReleaseDate.Format(time.DateOnly))
	}
	if digest := fctl.StringPointerToString(version.Digest); digest != "" {
		details = append(details, fctl.Truncate(digest, 20))
	}
	if len(details) == 0 {
		return latest
	}
	return latest + " (" + strings.Join(details, ", ") + ")"
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
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
