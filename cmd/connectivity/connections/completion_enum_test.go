package connections

import (
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

// prereleaseCatalog serves a connector with no stable release, like every
// connector published on the catalogue today.
type prereleaseCatalog struct {
	requested []string
}

func (c *prereleaseCatalog) RoundTrip(r *http.Request) (*http.Response, error) {
	c.requested = append(c.requested, r.URL.Path)
	switch r.URL.Path {
	case "/api/connectivity/connectors/ee-stripeconnect/versions/stable":
		return jsonHTTPResponse(r, http.StatusNotFound, `{"errorCode":"not_found","errorMessage":"no stable version"}`), nil
	case "/api/connectivity/connectors/ee-stripeconnect":
		return mockHTTPResponse(r, http.StatusOK, connectivity.Connector{Name: "ee-stripeconnect", LatestVersion: stringPtr("v0.5.0-beta.14")}, nil)
	case "/api/connectivity/connectors/ee-stripeconnect/versions/v0.5.0-beta.14":
		return mockHTTPResponse(r, http.StatusOK, connectivity.ConnectorVersion{Version: "v0.5.0-beta.14", ConfigSchema: map[string]any{
			"env": map[string]any{
				"required": []any{"ENVIRONMENT"},
				"properties": map[string]any{
					"ENVIRONMENT": map[string]any{"type": "string", "enum": []any{"sandbox", "live"}, "description": "Bound Stripe environment: sandbox or live. Every object is checked."},
				},
			},
		}}, nil)
	}
	return jsonHTTPResponse(r, http.StatusNotFound, `{}`), nil
}

func installCompletionCommand(t *testing.T, flags ...string) *cobra.Command {
	t.Helper()
	command := NewInstallCommand(mockReadFile(nil), mockPathCompleter(nil))
	require.NoError(t, command.ParseFlags(flags))
	command.SetContext(t.Context())
	return command
}

func TestInstallSetCompletionGivenNoStableReleaseThenUsesTheLatestVersionSchema(t *testing.T) {
	catalog := &prereleaseCatalog{}
	command := installCompletionCommand(t)

	candidates, directive := completeSetValuesWithClient(mockSDK(catalog), command, []string{"ee-stripeconnect"}, "", resolveInstallConnectorVersion, nil)

	require.Equal(t, []string{"ENVIRONMENT=\trequired · Bound Stripe environment: sandbox or live. [sandbox | live]"}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)
	require.Contains(t, catalog.requested, "/api/connectivity/connectors/ee-stripeconnect")
}

func TestInstallSetCompletionGivenAServerErrorThenKeepsTheStableSelection(t *testing.T) {
	failing := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/connectivity/connectors/ee-stripeconnect/versions/stable" {
			return jsonHTTPResponse(r, http.StatusInternalServerError, `{"errorCode":"internal","errorMessage":"unavailable"}`), nil
		}
		t.Errorf("unexpected fallback request %s", r.URL.Path)
		return jsonHTTPResponse(r, http.StatusNotFound, `{}`), nil
	})

	candidates, _ := completeSetValuesWithClient(mockSDK(failing), installCompletionCommand(t), []string{"ee-stripeconnect"}, "", resolveInstallConnectorVersion, nil)

	require.Empty(t, candidates, "a transient error must not switch completion to another schema")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInstallSetCompletionGivenAnExplicitChannelThenDoesNotFallBack(t *testing.T) {
	catalog := &prereleaseCatalog{}
	command := installCompletionCommand(t, "--channel", "stable")

	candidates, _ := completeSetValuesWithClient(mockSDK(catalog), command, []string{"ee-stripeconnect"}, "", resolveInstallConnectorVersion, nil)

	require.Empty(t, candidates)
	require.NotContains(t, catalog.requested, "/api/connectivity/connectors/ee-stripeconnect")
}

func TestSetCompletionOffersEnumValuesAfterTheKey(t *testing.T) {
	catalog := &prereleaseCatalog{}
	command := installCompletionCommand(t)

	candidates, directive := completeSetValuesWithClient(mockSDK(catalog), command, []string{"ee-stripeconnect"}, "ENVIRONMENT=s", resolveInstallConnectorVersion, nil)

	require.Equal(t, []string{"ENVIRONMENT=sandbox"}, candidates)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveKeepOrder, directive)
}

func TestSchemaFieldDescriptionFallsBackToTheFieldKind(t *testing.T) {
	require.Equal(t, "file configuration", schemaFieldDescription(configuration.SchemaField{Kind: configuration.ConfigFile}))
	require.Equal(t, "required · environment configuration", schemaFieldDescription(configuration.SchemaField{Kind: configuration.ConfigEnv, Required: true}))
}

func TestVersionDescriptionMarksLatestAndShortensTheDigest(t *testing.T) {
	summary := connectivity.ConnectorVersionSummary{Version: "v1.0.0", Digest: stringPtr("sha256:8f67ebd216c89fb955e70a80c1f4bb462d6fd89ed3b1b9a0f432a51f3b589930")}

	require.Equal(t, "latest · sha256:8f67ebd216c8…", versionDescription(summary, true))
	require.Equal(t, "", versionDescription(connectivity.ConnectorVersionSummary{Version: "v1.0.0"}, false))
}

func TestResolveChannelVersionRejectsAPageWithoutData(t *testing.T) {
	missingData := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(r, http.StatusOK, `{"cursor":{"pageSize":100,"hasMore":false}}`), nil
	})

	_, err := resolveChannelVersion(t.Context(), mockSDK(missingData), "ee-stripeconnect", "beta", "")

	require.ErrorContains(t, err, "cursor.data must be an array")
}
