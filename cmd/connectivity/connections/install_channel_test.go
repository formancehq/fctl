package connections

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

// betaOnlyCatalog serves a connector whose stable channel is empty, as the API
// reports it, and records any connection install would create.
type betaOnlyCatalog struct {
	latest    string
	requested []string
	created   *connectivity.ConnectionCreate
}

func (c *betaOnlyCatalog) RoundTrip(r *http.Request) (*http.Response, error) {
	c.requested = append(c.requested, r.Method+" "+r.URL.Path)
	switch {
	case r.URL.Path == "/api/connectivity/connectors/ee-synthetic/versions/stable":
		return jsonHTTPResponse(r, http.StatusNotFound, `{"errorCode":"channel_empty","errorMessage":"no stable version"}`), nil
	case r.URL.Path == "/api/connectivity/connectors/ee-synthetic":
		return mockHTTPResponse(r, http.StatusOK, connectivity.Connector{Name: "ee-synthetic", LatestVersion: stringPtr(c.latest)}, nil)
	case r.URL.Path == "/api/connectivity/connectors/ee-synthetic/versions/beta":
		return mockHTTPResponse(r, http.StatusOK, connectivity.ConnectorVersion{Version: c.latest}, nil)
	case r.Method == http.MethodPost && r.URL.Path == "/api/connectivity/connections":
		var body connectivity.ConnectionCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		c.created = &body
		return mockHTTPResponse(r, http.StatusCreated, connectivity.Connection{Name: body.Name, ConnectionConfiguration: createdConfiguration(body.ConnectionConfiguration)}, nil)
	}
	return jsonHTTPResponse(r, http.StatusNotFound, `{"errorCode":"not_found"}`), nil
}

func TestInstallGivenNoStableReleaseWhenNoSelectorThenGuidesWithoutInstalling(t *testing.T) {
	catalog := &betaOnlyCatalog{latest: "v0.5.0-beta.14"}
	command := testNewInstallCommand(factoryReturning(catalog), mockReadFile(nil), mockPathCompleter(nil))
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{"ee-synthetic", "--ledger=demo", "--confirm"})

	err := command.Execute()

	require.NoError(t, err)
	require.Nil(t, catalog.created, "install must not pick a prerelease on the user's behalf")
	require.Empty(t, stdout.String(), "nothing installed, nothing on stdout")
	require.Len(t, strings.Split(strings.TrimSpace(stderr.String()), "\n"), 1, "one guidance line, no repeated API message")
}

func TestInstallGivenNoStableReleaseWhenJSONThenReportsGuidedWithoutAConnection(t *testing.T) {
	catalog := &betaOnlyCatalog{latest: "v0.5.0-beta.14"}
	command := testNewInstallCommand(factoryReturning(catalog), mockReadFile(nil), mockPathCompleter(nil))
	command.Flags().String(fctl.OutputFlag, "plain", "")
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{"ee-synthetic", "--ledger=demo", "--confirm", "--output=json"})

	require.NoError(t, command.Execute())

	var envelope struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	require.Equal(t, map[string]any{"guided": true}, envelope.Data)
}

func TestInstallGivenNoStableReleaseWhenChannelIsExplicitThenKeepsTheNativeError(t *testing.T) {
	catalog := &betaOnlyCatalog{latest: "v0.5.0-beta.14"}

	_, err := executeCommand(testNewInstallCommand(factoryReturning(catalog), mockReadFile(nil), mockPathCompleter(nil)), "ee-synthetic", "--ledger=demo", "--channel=stable", "--confirm")

	require.ErrorContains(t, err, "no stable version")
	require.Nil(t, catalog.created)
	require.NotContains(t, catalog.requested, "GET /api/connectivity/connectors/ee-synthetic")
}

func TestSelectorGuidanceNamesTheLatestVersionSelectors(t *testing.T) {
	catalog := &betaOnlyCatalog{latest: "v0.5.0-beta.14"}
	apiErr := &connectivity.Error{ErrorCode: stringPtr("channel_empty"), ErrorMessage: stringPtr("no stable version")}

	guidance := selectorGuidance(t.Context(), mockSDK(catalog), installCompletionCommand(t), "ee-synthetic", apiErr)

	require.Contains(t, guidance, "--channel beta")
	require.Contains(t, guidance, "--version v0.5.0-beta.14")
}

func TestSelectorGuidanceGivenAnUnchannelledLatestVersionThenStaysGeneric(t *testing.T) {
	catalog := &betaOnlyCatalog{latest: "v0.5.0-dev.3"}
	apiErr := &connectivity.Error{ErrorCode: stringPtr("channel_empty"), ErrorMessage: stringPtr("no stable version")}

	guidance := selectorGuidance(t.Context(), mockSDK(catalog), installCompletionCommand(t), "ee-synthetic", apiErr)

	require.NotEmpty(t, guidance)
	require.NotContains(t, guidance, "v0.5.0-dev.3")
}

func TestSelectorGuidanceGivenAnotherErrorOrAnExplicitChannelThenGivesNone(t *testing.T) {
	catalog := &betaOnlyCatalog{latest: "v0.5.0-beta.14"}

	require.Empty(t, selectorGuidance(t.Context(), mockSDK(catalog), installCompletionCommand(t), "ee-synthetic", &connectivity.Error{ErrorCode: stringPtr("internal")}))
	require.Empty(t, selectorGuidance(t.Context(), mockSDK(catalog), installCompletionCommand(t, "--channel", "stable"), "ee-synthetic", &connectivity.Error{ErrorCode: stringPtr("channel_empty")}))
}

func TestVersionChannel(t *testing.T) {
	tests := map[string]struct {
		channel string
		ok      bool
	}{
		"v1.2.0":          {"stable", true},
		"1.2.0":           {"stable", true},
		"v1.2.0-rc.1":     {"rc", true},
		"v0.5.0-beta.14":  {"beta", true},
		"v0.1.0-alpha":    {"alpha", true},
		"v0.1.0-dev.1":    {"", false},
		"v0.1.0-stable.1": {"", false},
		"":                {"", false},
		"latest":          {"", false},
	}
	for version, want := range tests {
		t.Run(version, func(t *testing.T) {
			channel, ok := versionChannel(version)
			require.Equal(t, want.ok, ok)
			require.Equal(t, want.channel, channel)
		})
	}
}
