package connectorinstances

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

func TestSchemaFieldsExtractsEnvironmentAndFileMetadata(t *testing.T) {
	fields, err := SchemaFields(versionWithFullSchema())

	require.NoError(t, err)
	require.Equal(t, SchemaField{
		Key:         "TOKEN",
		Kind:        ConfigEnv,
		Required:    true,
		Password:    true,
		Description: "Authentication token",
	}, fields["TOKEN"])
	require.Equal(t, SchemaField{
		Key:         "/etc/plugin/config.yaml",
		Kind:        ConfigFile,
		Required:    true,
		Description: "Connector configuration",
	}, fields["/etc/plugin/config.yaml"])
	require.False(t, fields["TIMEOUT"].Required)
	require.True(t, fields["/etc/plugin/ca.pem"].Password)
}

func TestSchemaFieldsRejectsAmbiguousAndMalformedSchemas(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]any
		want   string
	}{
		{
			name:   "malformed env section",
			schema: map[string]any{"env": "not-an-object"},
			want:   "env",
		},
		{
			name: "duplicate env and file key",
			schema: map[string]any{
				"env":   map[string]any{"properties": map[string]any{"SAME": map[string]any{}}},
				"files": map[string]any{"properties": map[string]any{"SAME": map[string]any{}}},
			},
			want: "SAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := &connectivityclient.ConnectorVersion{ConfigSchema: tt.schema}

			_, err := SchemaFields(version)

			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestSchemaFieldsSupportsLegacyFlatSchemaAndSecretMarker(t *testing.T) {
	version := &connectivityclient.ConnectorVersion{ConfigSchema: map[string]any{
		"type":     "object",
		"required": []any{"LEGACY_TOKEN"},
		"properties": map[string]any{
			"LEGACY_TOKEN": map[string]any{
				"type":        "string",
				"x-secret":    true,
				"description": "Legacy token",
			},
		},
	}}

	fields, err := SchemaFields(version)

	require.NoError(t, err)
	require.Equal(t, SchemaField{
		Key:         "LEGACY_TOKEN",
		Kind:        ConfigEnv,
		Required:    true,
		Password:    true,
		Description: "Legacy token",
	}, fields["LEGACY_TOKEN"])
}

func TestBuildInstallConfigAllowsConnectorVersionWithoutSchema(t *testing.T) {
	version := &connectivityclient.ConnectorVersion{Version: "2.0.0", Image: "example/connector:2.0.0"}

	got, err := BuildInstallConfig(&cobra.Command{}, version, InputOptions{}, mapReadFile(nil))

	require.NoError(t, err)
	require.Empty(t, got.Env)
	require.Empty(t, got.Files)
}

func TestBuildInstallConfigParsesFlatYAMLAsEnvironmentValues(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{
		"config.yaml": "API_URL: https://config.example\nTOKEN: config-token\nTIMEOUT: 45s\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "https://config.example", *got.Env["API_URL"].Value)
	require.Equal(t, "config-token", *got.Env["TOKEN"].Value)
	require.Equal(t, "45s", *got.Env["TIMEOUT"].Value)
}

func TestBuildInstallConfigParsesStructuredYAML(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{
		"config.yaml": `env:
  API_URL:
    value: https://structured.example
  TOKEN:
    secretRef:
      name: connector-secrets
      key: token
files:
  - path: /etc/plugin/config.yaml
    configMapRef:
      name: connector-config
      key: config.yaml
  - path: /etc/plugin/ca.pem
    value: structured-ca
`,
	}))

	require.NoError(t, err)
	require.Equal(t, "https://structured.example", *got.Env["API_URL"].Value)
	require.Equal(t, &connectivityclient.KeyRef{Name: "connector-secrets", Key: "token"}, got.Env["TOKEN"].SecretRef)
	require.Nil(t, got.Env["TOKEN"].Value)
	require.Equal(t, &connectivityclient.KeyRef{Name: "connector-config", Key: "config.yaml"}, got.Files[0].ConfigMapRef)
	require.Equal(t, "structured-ca", *got.Files[1].Value)
}

func TestBuildInstallConfigParsesDotenvFilesInOrder(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"first.env", "second.env"}}, mapReadFile(map[string]string{
		"first.env":  "# comment\n\nAPI_URL=https://first.example\nTOKEN=first=token\n",
		"second.env": "API_URL=https://second.example\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "https://second.example", *got.Env["API_URL"].Value)
	require.Equal(t, "first=token", *got.Env["TOKEN"].Value)
}

func TestBuildInstallConfigParsesEstablishedDotenvSyntax(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"values.env"}}, mapReadFile(map[string]string{
		"values.env": ` export API_URL = https://env.example # endpoint
TOKEN="line-one\nline-two\t\"quoted\"\\tail"
TIMEOUT='  literal # value  '
`,
	}))

	require.NoError(t, err)
	require.Equal(t, "https://env.example", *got.Env["API_URL"].Value)
	require.Equal(t, "line-one\nline-two\t\"quoted\"\\tail", *got.Env["TOKEN"].Value)
	require.Equal(t, "  literal # value  ", *got.Env["TIMEOUT"].Value)
}

func TestBuildInstallConfigUnquotesDotenvValueBeforeTrailingComment(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"values.env"}}, mapReadFile(map[string]string{
		"values.env": "TOKEN=\"abc#def\" # trailing comment\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "abc#def", *got.Env["TOKEN"].Value)
}

func TestBuildInstallConfigPreservesHashWithinUnquotedDotenvValue(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"values.env"}}, mapReadFile(map[string]string{
		"values.env": "TOKEN=abc#def\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "abc#def", *got.Env["TOKEN"].Value)
}

func TestBuildInstallConfigAppliesDocumentedSourcePrecedence(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{
		ConfigFile: "config.yaml",
		EnvFiles:   []string{"first.env", "second.env"},
		SetValues:  []string{"API_URL=https://set-one.example", "API_URL=https://set-two.example"},
	}, mapReadFile(map[string]string{
		"config.yaml": "API_URL: https://config.example\n",
		"first.env":   "API_URL=https://env-one.example\n",
		"second.env":  "API_URL=https://env-two.example\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "https://set-two.example", *got.Env["API_URL"].Value)
	require.NotContains(t, got.Env, "TOKEN", "no source supplied TOKEN")
}

func TestBuildInstallConfigReadsInlineFileAndStdinSetValues(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{SetValues: []string{
		"TIMEOUT=90s",
		"TOKEN=@token.txt",
		"/etc/plugin/config.yaml=@-",
	}}, mapReadFile(map[string]string{
		"token.txt": "file-token\n",
		"-":         "stdin-config\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "90s", *got.Env["TIMEOUT"].Value)
	require.Equal(t, "file-token\n", *got.Env["TOKEN"].Value)
	require.Equal(t, "stdin-config\n", *got.Files[0].Value)
}

func TestBuildInstallConfigBuildsSecretAndConfigMapReferences(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{SetValues: []string{
		"TOKEN=secret://connector-secrets/token",
		"/etc/plugin/config.yaml=configmap://connector-config/config.yaml",
	}}, mapReadFile(nil))

	require.NoError(t, err)
	require.Equal(t, &connectivityclient.KeyRef{Name: "connector-secrets", Key: "token"}, got.Env["TOKEN"].SecretRef)
	require.Nil(t, got.Env["TOKEN"].Value)
	require.Equal(t, &connectivityclient.KeyRef{Name: "connector-config", Key: "config.yaml"}, got.Files[0].ConfigMapRef)
	require.Nil(t, got.Files[0].Value)
}

func TestBuildInstallConfigRoutesAssignmentsUsingSchema(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{SetValues: []string{
		"API_URL=https://routed.example",
		"/etc/plugin/ca.pem=routed-ca",
	}}, mapReadFile(nil))

	require.NoError(t, err)
	require.Equal(t, "https://routed.example", *got.Env["API_URL"].Value)
	require.Equal(t, "/etc/plugin/ca.pem", got.Files[0].Path)
	require.Equal(t, "routed-ca", *got.Files[0].Value)
}

func TestBuildInstallConfigReportsSortedMissingRequiredKeys(t *testing.T) {
	_, err := BuildInstallConfig(&cobra.Command{}, versionWithFullSchema(), InputOptions{}, mapReadFile(nil))

	require.EqualError(t, err, "missing required configuration keys: /etc/plugin/config.yaml, API_URL, TOKEN")
}

func TestBuildInstallConfigReportsSortedUnknownKeys(t *testing.T) {
	_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{SetValues: []string{
		"WRONG=value",
		"BOGUS=value",
	}}, mapReadFile(nil))

	require.EqualError(t, err, "unknown configuration keys: BOGUS, WRONG")
}

func TestBuildInstallConfigRejectsMalformedAssignmentsAndReferences(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "assignment without equals", value: "TOKEN", want: "KEY=value"},
		{name: "empty key", value: "=value", want: "empty key"},
		{name: "secret without key", value: "TOKEN=secret://name", want: "secret reference"},
		{name: "secret without name", value: "TOKEN=secret:///key", want: "secret reference"},
		{name: "configmap without key", value: "TOKEN=configmap://name/", want: "configmap reference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{SetValues: []string{tt.value}}, mapReadFile(nil))

			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestBuildInstallConfigMalformedSetErrorDoesNotEchoSecretAssignment(t *testing.T) {
	const secretSentinel = "SECRET-SENTINEL-MUST-NOT-LEAK"
	tests := []string{
		"TOKEN-" + secretSentinel,
		"=" + secretSentinel,
	}

	for _, assignment := range tests {
		_, err := BuildInstallConfig(
			&cobra.Command{},
			versionWithOptionalSchema(),
			InputOptions{SetValues: []string{assignment}},
			mapReadFile(nil),
		)

		require.Error(t, err)
		require.NotContains(t, err.Error(), assignment)
		require.NotContains(t, err.Error(), secretSentinel)
	}
}

func TestBuildInstallConfigReturnsInputReadAndParseErrors(t *testing.T) {
	t.Run("unreadable config", func(t *testing.T) {
		_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "missing.yaml"}, mapReadFile(nil))

		require.ErrorContains(t, err, "missing.yaml")
	})

	t.Run("malformed config", func(t *testing.T) {
		_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{"config.yaml": "env: ["}))

		require.ErrorContains(t, err, "config.yaml")
	})

	t.Run("malformed dotenv", func(t *testing.T) {
		_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"bad.env"}}, mapReadFile(map[string]string{"bad.env": "TOKEN\n"}))

		require.ErrorContains(t, err, "bad.env:1")
	})
}

func TestBuildConfigureConfigPreservesUntouchedFiles(t *testing.T) {
	mode := int32(0o600)
	current := &connectivityclient.ConnectorInstanceConfig{Files: []connectivityclient.FileMount{
		{Path: "/etc/a", Value: stringPtr("a"), Mode: &mode},
		{Path: "/etc/b", Value: stringPtr("b")},
	}}
	got, err := BuildConfigureConfig(&cobra.Command{}, versionWithFileSchema(), current,
		InputOptions{SetValues: []string{"/etc/a=@new"}},
		func(_ *cobra.Command, name string) (string, error) {
			require.Equal(t, "new", name)
			return "changed", nil
		})

	require.NoError(t, err)
	require.Len(t, got.Files, 2)
	require.Equal(t, "changed", *got.Files[0].Value)
	require.NotNil(t, got.Files[0].Mode)
	require.Equal(t, int32(0o600), *got.Files[0].Mode)
	require.Equal(t, "b", *got.Files[1].Value)
}

func TestBuildInstallConfigRejectsStructuredFileWithoutPath(t *testing.T) {
	_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{
		"config.yaml": "files:\n  - value: missing-path\n",
	}))

	require.ErrorContains(t, err, "file path is required")
}

func TestBuildConfigureConfigDeepCopiesCurrentConfiguration(t *testing.T) {
	current := &connectivityclient.ConnectorInstanceConfig{
		Env: map[string]connectivityclient.EnvValue{
			"API_URL": {Value: stringPtr("https://current.example")},
		},
		Files: []connectivityclient.FileMount{
			{Path: "/etc/a", Value: stringPtr("current-a")},
		},
	}

	got, err := BuildConfigureConfig(&cobra.Command{}, versionWithFileSchema(), current, InputOptions{
		SetValues: []string{"API_URL=https://changed.example", "/etc/a=changed-a"},
	}, mapReadFile(nil))

	require.NoError(t, err)
	require.Equal(t, "https://changed.example", *got.Env["API_URL"].Value)
	require.Equal(t, "changed-a", *got.Files[0].Value)
	require.Equal(t, "https://current.example", *current.Env["API_URL"].Value)
	require.Equal(t, "current-a", *current.Files[0].Value)
}
