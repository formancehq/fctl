package connectivityclient

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigurationUsesLiteralPublicFields(t *testing.T) {
	c := ConnectorInstanceConfiguration{Connector: "stripe", Ledger: "main", Version: pointer("v1.0.0"), Suspend: pointer(false), InitialCursor: []byte{0, 0, 0, 0, 0, 0, 3, 232}, Env: map[string]string{"EMPTY": "", "SETTING": "value"}, Files: map[string]string{"/etc/plugin/key": "contents"}}
	body, err := json.Marshal(c)
	require.NoError(t, err)
	require.JSONEq(t, `{"connector":"stripe","ledger":"main","version":"v1.0.0","suspend":false,"initialCursor":"AAAAAAAAA+g=","env":{"EMPTY":"","SETTING":"value"},"files":{"/etc/plugin/key":"contents"}}`, string(body))
	require.Equal(t, "stripe", c.ConnectorName())
	require.Equal(t, &ConnectorInstanceConfig{Env: c.Env, Files: c.Files}, c.Configuration())
	require.Empty(t, (ConnectorInstanceConfiguration{}).ConnectorName())
	for _, cursor := range [][]byte{nil, {}} {
		body, err = json.Marshal(ConnectorInstanceConfiguration{Ledger: "main", InitialCursor: cursor})
		require.NoError(t, err)
		require.JSONEq(t, `{"ledger":"main"}`, string(body))
	}
}

func TestDecodeFlatConnectorAndVersion(t *testing.T) {
	var c Connector
	err := decodeResponse(strings.NewReader(`{"name":"stripe","createdAt":"2026-10-28T10:00:00Z","displayName":"Stripe","description":"Payments","catalog":"providers","latestVersion":"v2.0.0","tagline":"Simple payments","tags":["provider:psp"],"branding":{"displayName":"Stripe Inc.","accentColor":"#635bff","logoSvg":"<svg/>","logoSvgDark":"<svg id='dark'/>"}}`), &c, true)
	require.NoError(t, err)
	require.Equal(t, "stripe", c.Name)
	require.NotNil(t, c.CreatedAt)
	require.Equal(t, "Stripe", *c.DisplayName)
	require.Equal(t, "Payments", *c.Description)
	require.Equal(t, "providers", *c.Catalog)
	require.Equal(t, "v2.0.0", *c.LatestVersion)
	require.Equal(t, "Simple payments", *c.Tagline)
	require.Equal(t, []string{"provider:psp"}, c.Tags)
	require.Equal(t, "Stripe Inc.", *c.Branding.DisplayName)
	require.Equal(t, "#635bff", *c.Branding.AccentColor)
	require.Equal(t, "<svg/>", *c.Branding.LogoSvg)
	require.Equal(t, "<svg id='dark'/>", *c.Branding.LogoSvgDark)
	var version ConnectorVersion
	err = decodeResponse(strings.NewReader(`{"version":"v2.0.0","digest":"sha256:abc","releaseDate":"2026-10-28T10:00:00Z","configSchema":{"type":"object"},"additionalMetadata":{"feature":true}}`), &version, true)
	require.NoError(t, err)
	require.NotNil(t, version.ReleaseDate)
	require.Equal(t, "sha256:abc", *version.Digest)
	require.Equal(t, "object", version.ConfigSchema["type"])
	require.Equal(t, true, version.AdditionalMetadata["feature"])
}

func TestInstanceResponseAllowsAbsentStatusAndStoredUnboundConfiguration(t *testing.T) {
	for _, status := range []string{"", `,"status":null`, `,"status":{"lifecycle":"pending","suspendedBy":[]}`} {
		var instance ConnectorInstance
		require.NoError(t, decodeResponse(strings.NewReader(`{"name":"unbound","configuration":{"ledger":"main"}`+status+`}`), &instance, true))
		require.Empty(t, instance.Configuration.Connector)
	}
}

func TestInstanceStatusDecodesRecordedOutcomes(t *testing.T) {
	var instance ConnectorInstance
	err := decodeResponse(strings.NewReader(`{"name":"worker","configuration":{"connector":"stripe","ledger":"main","channel":"stable","suspend":false},"status":{"appliedVersion":"v1.0.0","lifecycle":"suspended","ingestion":{"state":"paused","progress":{"current":0},"lastError":"historical retry"},"error":{"code":"failure_unknown","message":"Recorded failure"},"suspendedBy":[{"source":"request","name":"worker"},{"source":"policy","name":"maintenance"}]}}`), &instance, true)
	require.NoError(t, err)
	require.False(t, *instance.Configuration.Suspend)
	require.Equal(t, "v1.0.0", *instance.Status.AppliedVersion)
	require.Equal(t, "suspended", instance.Status.Lifecycle)
	require.Equal(t, "paused", *instance.Status.Ingestion.State)
	require.NotNil(t, instance.Status.Ingestion.Progress)
	require.Zero(t, instance.Status.Ingestion.Progress.Current)
	require.Equal(t, "historical retry", *instance.Status.Ingestion.LastError)
	require.Equal(t, &StatusError{Code: "failure_unknown", Message: "Recorded failure"}, instance.Status.Error)
	require.Equal(t, []SuspensionSource{{Source: "request", Name: "worker"}, {Source: "policy", Name: "maintenance"}}, instance.Status.SuspendedBy)
}

func TestInvalidInstanceResponsesRejectEntirePage(t *testing.T) {
	for _, body := range []string{
		`{}`, `{"name":"worker"}`, `{"name":"worker","configuration":null}`, `{"name":"worker","configuration":{}}`,
		`{"metadata":{"name":"worker"},"spec":{"ledger":"main"}}`,
		`{"name":"worker","configuration":{"ledger":"main","initialCursor":"not-base64!"}}`,
		`{"name":"worker","configuration":{"ledger":"main","env":{"TOKEN":{"value":"secret"}}}}`,
		`{"name":"worker","configuration":{"ledger":"main","files":[{"path":"/etc/key","value":"secret"}]}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"failed","suspendedBy":[],"error":{"code":"failure_unknown"}}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"suspended","suspendedBy":[{"source":"request"}]}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"Running","suspendedBy":[]}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active"}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active","suspendedBy":[],"ingestion":{"state":"bogus"}}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active","suspendedBy":[],"ingestion":{"progress":{}}}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active","suspendedBy":[],"error":{"code":"bogus","message":"bad"}}}`,
		`{"name":"worker","configuration":{"ledger":"main"},"status":{"lifecycle":"active","suspendedBy":[{"source":"Policy","name":"maintenance"}]}}`,
	} {
		t.Run(body, func(t *testing.T) {
			var instance ConnectorInstance
			require.Error(t, decodeResponse(strings.NewReader(body), &instance, true))
			c := New("https://stack.example", responseClient(200, `{"cursor":{"data":[`+instanceResponse+`,`+body+`]}}`))
			page, err := c.ListConnectorInstances(t.Context(), ListOptions{})
			require.Error(t, err)
			require.Nil(t, page)
		})
	}
}

func TestPublicInstanceResponseDecodesOpaqueCursorBytes(t *testing.T) {
	var instance ConnectorInstance
	err := decodeResponse(strings.NewReader(`{"name":"worker","configuration":{"connector":"stripe","ledger":"main","version":"v1.0.0","initialCursor":"AAAAAAAAA+g="}}`), &instance, true)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 3, 232}, instance.Configuration.InitialCursor)
}
