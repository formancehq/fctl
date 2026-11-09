package connectors

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

// groupedSchema mirrors the shape catalog connectors publish: env and files
// sections plus presentation groups that order fields and flag advanced ones.
func groupedSchema() map[string]any {
	return map[string]any{
		"env": map[string]any{
			"required": []any{"SECRET_KEY", "ENVIRONMENT"},
			"properties": map[string]any{
				"SECRET_KEY":  map[string]any{"type": "string", "format": "password", "description": "Stripe restricted key for the platform. Rotate it yearly."},
				"ENVIRONMENT": map[string]any{"type": "string", "description": "Bound Stripe environment: sandbox or live."},
				"MAX_RETRIES": map[string]any{"type": "integer", "default": 4, "description": "Retries per request."},
				"UNGROUPED":   map[string]any{"type": "string", "description": "Not listed in any group."},
			},
		},
		"files": map[string]any{
			"properties": map[string]any{"/var/run/secrets/stripe/ca.pem": map[string]any{"type": "string", "description": "Custom CA bundle."}},
		},
		"presentation": map[string]any{"groups": []any{
			map[string]any{"title": "Stripe environment", "properties": []any{"env.ENVIRONMENT", "files/var/run/secrets/stripe/ca.pem"}},
			map[string]any{"title": "Authentication", "properties": []any{"env.SECRET_KEY"}},
			map[string]any{"title": "Limits", "advanced": true, "properties": []any{"env.MAX_RETRIES"}},
		}},
		"requirements": []any{
			map[string]any{"anyOf": []any{"env.SECRET_KEY"}, "description": "Provide the secret key."},
			map[string]any{"anyOf": []any{"env.SECRET_KEY"}, "description": "Provide the secret key."},
		},
	}
}

func showWithSchema(t *testing.T, schema map[string]any) string {
	t.Helper()
	connector := connectorFixture("ee-stripeconnect")
	connector.Description = fctl.Ptr(strings.Repeat("Ingest connected-account topology and balance journals. ", 4))
	released := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)
	client := connectorTransportMock{
		get: func(context.Context, string) (*connectivity.Connector, error) { return &connector, nil },
		listVersions: func(context.Context, string, connectivityinternal.ListOptions) (*connectorVersionPage, error) {
			return versionsFixture(), nil
		},
		getVersion: func(context.Context, string, string) (*connectivity.ConnectorVersion, error) {
			return &connectivity.ConnectorVersion{Version: "2.0.0", ReleaseDate: &released, Digest: fctl.Ptr("sha256:70b21b53a6920c8ea1f2"), ConfigSchema: schema}, nil
		},
	}
	output, err := executeCommand(testNewShowCommand(sdkFor(client)), "ee-stripeconnect")
	require.NoError(t, err)
	return pterm.RemoveColorFromString(output)
}

func TestShowConnectorGroupsFieldsInPresentationOrder(t *testing.T) {
	output := showWithSchema(t, groupedSchema())

	order := []string{"— Stripe environment", "ENVIRONMENT", "/var/run/secrets/stripe/ca.pem", "— Authentication", "SECRET_KEY", "— Limits (advanced)", "MAX_RETRIES", "— Other", "UNGROUPED"}
	position := -1
	for _, value := range order {
		next := strings.Index(output, value)
		require.Greater(t, next, position, "%q out of order in:\n%s", value, output)
		position = next
	}
	require.Contains(t, output, "Configuration (2.0.0) · 5 fields, 2 required")
	require.Contains(t, output, "Latest Version     | 2.0.0 (released 2026-10-21, sha256:70b21b53a692…)")
}

func TestShowConnectorSummarizesDescriptionsDefaultsAndSecrets(t *testing.T) {
	output := showWithSchema(t, groupedSchema())

	require.Contains(t, output, "Stripe restricted key for the platform.")
	require.NotContains(t, output, "Rotate it yearly", "only the first sentence is shown; -o json keeps the rest")
	require.Contains(t, output, "Retries per request. Default: 4.")
	require.Regexp(t, `SECRET_KEY\s+\| yes \| secret`, output)
	require.Equal(t, 1, strings.Count(output, "- Provide the secret key."), "duplicate rules are printed once")
}

func TestShowConnectorWithoutPresentationKeepsSourceAndKeyOrder(t *testing.T) {
	schema := groupedSchema()
	delete(schema, "presentation")

	output := showWithSchema(t, schema)

	require.NotContains(t, output, "— ")
	require.Less(t, strings.Index(output, "ENVIRONMENT"), strings.Index(output, "UNGROUPED"))
	require.Less(t, strings.Index(output, "UNGROUPED"), strings.Index(output, "/var/run/secrets/stripe/ca.pem"))
}

func TestShowConnectorFitsAnEightyColumnTerminal(t *testing.T) {
	t.Cleanup(fctl.SetTerminalWidth(80))

	output := showWithSchema(t, groupedSchema())

	for _, line := range strings.Split(output, "\n") {
		require.LessOrEqual(t, utf8.RuneCountInString(strings.TrimRight(line, " ")), 80, "line %q", line)
	}
	require.Contains(t, output, "/var/run/secrets/stripe/ca.pem", "keys are never cut")
}

func TestListConnectorsCutsOnlyTheTaglineToFit(t *testing.T) {
	t.Cleanup(fctl.SetTerminalWidth(50))
	connector := connectorFixture("ee-adyenplatforms")
	connector.LatestVersion = fctl.Ptr("v0.5.0-beta.7")
	connector.Tagline = fctl.Ptr("Platform and marketplace payments on Adyen, tracked by balance account")
	client := connectorTransportMock{list: func(context.Context, connectivityinternal.ListOptions) (*connectorPage, error) {
		return &connectorPage{Items: []connectivity.Connector{connector}, PageSize: 15}, nil
	}}

	output, err := executeCommand(testNewListCommand(sdkFor(client)))

	require.NoError(t, err)
	output = pterm.RemoveColorFromString(output)
	require.Contains(t, output, "ee-adyenplatforms | v0.5.0-beta.7  | Platform and…")
	require.NotContains(t, output, "Next page", "no cursor line without more results")
}

func TestSchemaFieldsIncludesFieldsDeclaredThroughComposition(t *testing.T) {
	schema := map[string]any{
		"env": map[string]any{
			"allOf": []any{
				map[string]any{"required": []any{"API_KEY"}, "properties": map[string]any{"API_KEY": map[string]any{"type": "string", "format": "password"}, "RETRIES": map[string]any{"type": "integer", "default": 4}}},
				map[string]any{"$ref": "#/$defs/region"},
			},
			// References resolve within the section, as install and configure do.
			"$defs": map[string]any{"region": map[string]any{"properties": map[string]any{"REGION": map[string]any{"type": "string"}}}},
		},
	}

	fields := schemaFields(schema)

	keys := map[string]schemaField{}
	for _, field := range fields {
		keys[field.key] = field
	}
	require.Len(t, fields, 3)
	require.True(t, keys["API_KEY"].required)
	require.Equal(t, "integer", keys["RETRIES"].kind, "composed fields keep their type")
	require.Contains(t, keys["RETRIES"].description, "Default: 4")
	require.Equal(t, "secret", keys["API_KEY"].kind)
	require.Contains(t, keys, "REGION")
}

func TestSchemaFieldsUseResolvedMetadataForDirectlyDeclaredProperties(t *testing.T) {
	schema := map[string]any{
		"env": map[string]any{
			"properties": map[string]any{"API_KEY": map[string]any{"$ref": "#/$defs/secret"}},
			"$defs": map[string]any{"secret": map[string]any{
				"type": "string", "format": "password", "description": "Platform key. Rotate yearly.", "default": "unset",
			}},
		},
	}

	fields := schemaFields(schema)

	require.Len(t, fields, 1)
	require.Equal(t, "secret", fields[0].kind)
	require.Contains(t, fields[0].description, "Platform key.")
	require.Contains(t, fields[0].description, "Default: unset")
}
