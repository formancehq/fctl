package connectors

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
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
		fctl.WithShortDescription("Show a Connectivity connector"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithValidArgsFunction(CompleteConnectorNames()),
		fctl.WithController[*ShowStore](controller),
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
	if response == nil {
		return nil, fmt.Errorf("show connectivity connector %q: empty response", name)
	}
	if err := connectivityinternal.ResponseError(response.StatusCode, response.Error, response.RawResponse); err != nil {
		return nil, err
	}
	if response.Connector == nil {
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
			if response == nil {
				return nil, false, "", fmt.Errorf("show connectivity connector %q: empty version list response", name)
			}
			if err := connectivityinternal.ResponseError(response.StatusCode, response.Error, response.RawResponse); err != nil {
				return nil, false, "", err
			}
			if response.ConnectorVersionCursor == nil {
				return nil, false, "", fmt.Errorf("show connectivity connector %q: empty version list response", name)
			}
			page := response.ConnectorVersionCursor.Cursor
			if page.Data == nil {
				return nil, false, "", fmt.Errorf("show connectivity connector %q: version cursor.data must be an array", name)
			}
			return page.Data, page.HasMore, stringValue(page.Next), nil
		})
	if err != nil {
		return nil, err
	}
	c.store.Versions = versions

	// The server resolves the `latest` alias to the newest Validated version;
	// published-but-unresolvable catalogues degrade to the no-version notice.
	if len(versions) > 0 {
		response, err := client.GetConnectorVersion(cmd.Context(), operations.GetConnectorVersionRequest{Name: url.PathEscape(name), Version: connectivityinternal.VersionAliasLatest}, nil)
		if err == nil && response != nil {
			err = connectivityinternal.ResponseError(response.StatusCode, response.Error, response.RawResponse)
		}
		var sdkErr *sdkerrors.SDKError
		switch {
		case errors.As(err, &sdkErr) && sdkErr.StatusCode == http.StatusNotFound:
		case err != nil:
			return nil, err
		case response == nil || response.ConnectorVersion == nil:
			return nil, fmt.Errorf("show connectivity connector %q: empty latest version response", name)
		default:
			c.store.Version = response.ConnectorVersion
		}
	}
	return c, nil
}

func (c *ShowController) Render(cmd *cobra.Command, _ []string) error {
	connector := c.store.Connector
	out := cmd.OutOrStdout()

	fctl.Section.WithWriter(out).Println("Information")
	information := pterm.TableData{
		{pterm.LightCyan("Name"), connector.Name},
		{pterm.LightCyan("Created At"), timeValue(connector.CreatedAt)},
		{pterm.LightCyan("Display Name"), stringValue(connector.DisplayName)},
		{pterm.LightCyan("Description"), stringValue(connector.Description)},
		{pterm.LightCyan("Catalog"), stringValue(connector.Catalog)},
		{pterm.LightCyan("Tags"), strings.Join(connector.Tags, ", ")},
		{pterm.LightCyan("Tagline"), stringValue(connector.Tagline)},
		{pterm.LightCyan("Latest Version"), stringValue(connector.LatestVersion)},
	}
	if err := pterm.DefaultTable.WithWriter(out).WithData(information).Render(); err != nil {
		return err
	}

	if len(c.store.Versions) > 0 {
		fctl.Section.WithWriter(out).Println("Versions")
		versions := fctl.Map(c.store.Versions, func(version connectivity.ConnectorVersionSummary) []string {
			return []string{version.Version, stringValue(version.Digest), timeValue(version.ReleaseDate)}
		})
		versions = fctl.Prepend(versions, []string{"Version", "Digest", "Released"})
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
