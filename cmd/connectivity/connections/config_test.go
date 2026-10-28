package connections

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

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

func TestSchemaFieldsRejectsFlatSchemas(t *testing.T) {
	for _, schema := range []map[string]any{
		{"properties": map[string]any{"TOKEN": map[string]any{}}},
		{"env": map[string]any{"properties": map[string]any{"TOKEN": map[string]any{}}}, "properties": map[string]any{"OLD": map[string]any{}}},
	} {
		_, err := SchemaFields(&connectivityclient.ConnectorVersion{ConfigSchema: schema})
		require.ErrorContains(t, err, "flat schemas are unsupported")
	}
}

func TestSchemaFieldsCollectsLocalReferencesAndCompositionsWithoutOverstatingRequiredKeys(t *testing.T) {
	version := &connectivityclient.ConnectorVersion{ConfigSchema: map[string]any{
		"env": map[string]any{
			"$defs": map[string]any{
				"token": map[string]any{"type": "string", "format": "password", "description": "API token"},
			},
			"allOf": []any{
				map[string]any{"properties": map[string]any{"TOKEN": map[string]any{"$ref": "#/$defs/token"}}, "required": []any{"TOKEN"}},
				map[string]any{"anyOf": []any{
					map[string]any{"properties": map[string]any{"CLIENT_ID": map[string]any{"type": "string"}}, "required": []any{"CLIENT_ID"}},
					map[string]any{"properties": map[string]any{"CLIENT_SECRET": map[string]any{"type": "string"}}, "required": []any{"CLIENT_SECRET"}},
				}},
			},
		},
	}}

	fields, err := SchemaFields(version)

	require.NoError(t, err)
	require.True(t, fields["TOKEN"].Required)
	require.True(t, fields["TOKEN"].Password)
	require.Equal(t, "API token", fields["TOKEN"].Description)
	require.False(t, fields["CLIENT_ID"].Required, "either alternative can satisfy the schema")
	require.False(t, fields["CLIENT_SECRET"].Required, "either alternative can satisfy the schema")
}

func TestSchemaFieldsFollowsAnchoredDynamicAndRecursiveReferences(t *testing.T) {
	version := &connectivityclient.ConnectorVersion{ConfigSchema: map[string]any{
		"env": map[string]any{
			"$defs": map[string]any{
				"secret": map[string]any{"$dynamicAnchor": "secret", "format": "password", "description": "Anchored secret"},
			},
			"properties": map[string]any{
				"TOKEN": map[string]any{"$dynamicRef": "#secret"},
				"LOOP":  map[string]any{"$recursiveRef": "#"},
			},
		},
	}}

	fields, err := SchemaFields(version)

	require.NoError(t, err)
	require.True(t, fields["TOKEN"].Password)
	require.Equal(t, "Anchored secret", fields["TOKEN"].Description)
	require.Contains(t, fields, "LOOP", "recursive schemas are accepted and delegated to API validation")
}

func TestSchemaFieldsBoundsMetadataTraversalDepth(t *testing.T) {
	field := any(map[string]any{"format": "password"})
	for index := 0; index < maxSchemaCollectionDepth+1; index++ {
		field = map[string]any{"allOf": []any{field}}
	}
	version := &connectivityclient.ConnectorVersion{ConfigSchema: map[string]any{
		"env": map[string]any{"properties": map[string]any{"TOKEN": field}},
	}}

	_, err := SchemaFields(version)

	require.ErrorContains(t, err, "schema nesting exceeds")
}

func TestBuildInstallConfigDefersOpenAndPatternSchemaKeysToTheAPI(t *testing.T) {
	version := &connectivityclient.ConnectorVersion{ConfigSchema: map[string]any{
		"env": map[string]any{
			"type":              "object",
			"properties":        map[string]any{"KNOWN": map[string]any{"type": "string"}},
			"patternProperties": map[string]any{"^DYNAMIC_": map[string]any{"type": "string"}},
		},
	}}

	config, err := BuildInstallConfig(&cobra.Command{}, version, InputOptions{SetValues: []string{"DYNAMIC_TOKEN=value"}}, mapReadFile(nil))

	require.NoError(t, err)
	require.Equal(t, "value", config.Env["DYNAMIC_TOKEN"])
}

func TestBuildInstallConfigAllowsConnectorVersionWithoutSchema(t *testing.T) {
	version := &connectivityclient.ConnectorVersion{Version: "2.0.0"}

	got, err := BuildInstallConfig(&cobra.Command{}, version, InputOptions{}, mapReadFile(nil))

	require.NoError(t, err)
	require.Empty(t, got.Env)
	require.Empty(t, got.Files)
}

func TestBuildInstallConfigRejectsFlatYAML(t *testing.T) {
	_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{"config.yaml": "API_URL: https://config.example\n"}))
	require.ErrorContains(t, err, "unknown configuration keys: API_URL")
}

func TestBuildInstallConfigParsesLiteralYAML(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{"config.yaml": "env:\n  API_URL: https://structured.example\n  TOKEN: private-token\nfiles:\n  /etc/plugin/config.yaml: file-content\n  /etc/plugin/ca.pem: structured-ca\n"}))
	require.NoError(t, err)
	require.Equal(t, "https://structured.example", got.Env["API_URL"])
	require.Equal(t, "private-token", got.Env["TOKEN"])
	require.Equal(t, "file-content", got.Files["/etc/plugin/config.yaml"])
	require.Equal(t, "structured-ca", got.Files["/etc/plugin/ca.pem"])
}

func TestBuildInstallConfigParsesDotenvFilesInOrder(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"first.env", "second.env"}}, mapReadFile(map[string]string{
		"first.env":  "# comment\n\nAPI_URL=https://first.example\nTOKEN=first=token\n",
		"second.env": "API_URL=https://second.example\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "https://second.example", got.Env["API_URL"])
	require.Equal(t, "first=token", got.Env["TOKEN"])
}

func TestBuildInstallConfigParsesEstablishedDotenvSyntax(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"values.env"}}, mapReadFile(map[string]string{
		"values.env": ` export API_URL = https://env.example # endpoint
TOKEN="line-one\nline-two\t\"quoted\"\\tail"
TIMEOUT='  literal # value  '
`,
	}))

	require.NoError(t, err)
	require.Equal(t, "https://env.example", got.Env["API_URL"])
	require.Equal(t, "line-one\nline-two\t\"quoted\"\\tail", got.Env["TOKEN"])
	require.Equal(t, "  literal # value  ", got.Env["TIMEOUT"])
}

func TestBuildInstallConfigUnquotesDotenvValueBeforeTrailingComment(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"values.env"}}, mapReadFile(map[string]string{
		"values.env": "TOKEN=\"abc#def\" # trailing comment\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "abc#def", got.Env["TOKEN"])
}

func TestBuildInstallConfigPreservesHashWithinUnquotedDotenvValue(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{EnvFiles: []string{"values.env"}}, mapReadFile(map[string]string{
		"values.env": "TOKEN=abc#def\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "abc#def", got.Env["TOKEN"])
}

func TestBuildInstallConfigAppliesDocumentedSourcePrecedence(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{
		ConfigFile: "config.yaml",
		EnvFiles:   []string{"first.env", "second.env"},
		SetValues:  []string{"API_URL=https://set-one.example", "API_URL=https://set-two.example"},
	}, mapReadFile(map[string]string{
		"config.yaml": "env:\n  API_URL: https://config.example\n",
		"first.env":   "API_URL=https://env-one.example\n",
		"second.env":  "API_URL=https://env-two.example\n",
	}))

	require.NoError(t, err)
	require.Equal(t, "https://set-two.example", got.Env["API_URL"])
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
	require.Equal(t, "90s", got.Env["TIMEOUT"])
	require.Equal(t, "file-token\n", got.Env["TOKEN"])
	require.Equal(t, "stdin-config\n", got.Files["/etc/plugin/config.yaml"])
}

func TestBuildInstallConfigRejectsKubernetesReferences(t *testing.T) {
	for _, value := range []string{"secret://secret-name/key", "configmap://config-name/key"} {
		_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{SetValues: []string{"TOKEN=" + value}}, nil)
		require.ErrorContains(t, err, "Kubernetes references are unsupported")
		require.ErrorContains(t, err, "@file / @-")
		require.NotContains(t, err.Error(), value)
	}
}

func TestBuildInstallConfigRoutesAssignmentsUsingSchema(t *testing.T) {
	got, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{SetValues: []string{
		"API_URL=https://routed.example",
		"/etc/plugin/ca.pem=routed-ca",
	}}, mapReadFile(nil))

	require.NoError(t, err)
	require.Equal(t, "https://routed.example", got.Env["API_URL"])
	require.Equal(t, "routed-ca", got.Files["/etc/plugin/ca.pem"])
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
		{name: "secret without key", value: "TOKEN=secret://name", want: "Kubernetes references are unsupported"},
		{name: "secret without name", value: "TOKEN=secret:///key", want: "Kubernetes references are unsupported"},
		{name: "configmap without key", value: "TOKEN=configmap://name/", want: "Kubernetes references are unsupported"},
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

func TestBuildConfigDecodeErrorsDoNotEchoSecrets(t *testing.T) {
	const secretSentinel = "s3cr3t"
	tests := []struct {
		name     string
		contents string
	}{
		{name: "root scalar", contents: secretSentinel},
		{name: "environment sequence", contents: "env:\n  TOKEN: [" + secretSentinel + "]"},
		{name: "environment value sequence", contents: "env:\n  TOKEN:\n    value: [" + secretSentinel + "]"},
		{name: "secret reference scalar", contents: "env:\n  TOKEN:\n    secretRef: " + secretSentinel},
		{name: "configmap reference scalar", contents: "env:\n  TOKEN:\n    configMapRef: " + secretSentinel},
		{name: "file scalar", contents: "files:\n  - " + secretSentinel},
		{name: "file reference scalar", contents: "files:\n  - path: /etc/plugin/config.yaml\n    secretRef: " + secretSentinel},
		{name: "file mode scalar", contents: "files:\n  - path: /etc/plugin/config.yaml\n    value: content\n    mode: " + secretSentinel},
		{name: "duplicate key", contents: secretSentinel + ": first\n" + secretSentinel + ": second"},
		{name: "unknown alias", contents: "env: *" + secretSentinel},
		{name: "invalid tag", contents: "env: !!int " + secretSentinel},
		{name: "JSON environment scalar", contents: `{"env":{"TOKEN":["` + secretSentinel + `"]}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := InputOptions{ConfigFile: "config.yaml"}
			read := mapReadFile(map[string]string{"config.yaml": tt.contents})
			_, installErr := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), inputs, read)
			_, configureErr := BuildConfigureConfig(&cobra.Command{}, versionWithOptionalSchema(), nil, inputs, read)
			for _, err := range []error{installErr, configureErr} {
				require.Error(t, err)
				require.ErrorContains(t, err, "config.yaml")
				for cause := err; cause != nil; cause = errors.Unwrap(cause) {
					require.NotContains(t, cause.Error(), secretSentinel)
				}
				var typeErr *yaml.TypeError
				require.False(t, errors.As(err, &typeErr), "raw decoder errors must not remain recoverable")

			}
		})
	}
}

func TestBuildConfigDecodeErrorsRetainLineNumbers(t *testing.T) {
	_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{"config.yaml": "env: !!int s3cr3t"}))
	require.ErrorContains(t, err, "invalid YAML/JSON configuration")
	require.NotContains(t, err.Error(), "s3cr3t")
}

func TestBuildConfigureConfigPreservesUntouchedFiles(t *testing.T) {
	current := &connectivityclient.ConnectorInstanceConfig{Files: map[string]string{"/etc/a": "a", "/etc/b": "b"}}
	got, err := BuildConfigureConfig(&cobra.Command{}, versionWithFileSchema(), current, InputOptions{SetValues: []string{"/etc/a=@new"}}, mapReadFile(map[string]string{"new": "changed"}))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"/etc/a": "changed", "/etc/b": "b"}, got.Files)
	require.Equal(t, "a", current.Files["/etc/a"])
}

func TestBuildInstallConfigRejectsStructuredFileWithoutPath(t *testing.T) {
	_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), InputOptions{ConfigFile: "config.yaml"}, mapReadFile(map[string]string{
		"config.yaml": "files:\n  - value: missing-path\n",
	}))

	require.ErrorContains(t, err, "env and files must be mappings")
}

func TestBuildConfigureConfigDeepCopiesCurrentConfiguration(t *testing.T) {
	current := &connectivityclient.ConnectorInstanceConfig{
		Env: map[string]string{
			"API_URL": "https://current.example",
		},
		Files: map[string]string{"/etc/a": "current-a"},
	}

	got, err := BuildConfigureConfig(&cobra.Command{}, versionWithFileSchema(), current, InputOptions{
		SetValues: []string{"API_URL=https://changed.example", "/etc/a=changed-a"},
	}, mapReadFile(nil))

	require.NoError(t, err)
	require.Equal(t, "https://changed.example", got.Env["API_URL"])
	require.Equal(t, "changed-a", got.Files["/etc/a"])
	require.Equal(t, "https://current.example", current.Env["API_URL"])
	require.Equal(t, "current-a", current.Files["/etc/a"])
}

func TestConfigureRejectsUnrepresentableMapClearAndSetWithoutLosingIntent(t *testing.T) {
	_, err := BuildConfigureConfig(&cobra.Command{}, versionWithOptionalSchema(), nil,
		InputOptions{ConfigFile: "patch", SetValues: []string{"TOKEN=new-token"}, patch: map[string]any{}},
		mapReadFile(map[string]string{"patch": "env: null\n"}))
	require.ErrorContains(t, err, "use separate configure requests")
	require.NotContains(t, err.Error(), "new-token")
}

func TestConfigurationReadErrorsDoNotEchoSecretContents(t *testing.T) {
	for _, input := range []InputOptions{{ConfigFile: "config"}, {EnvFiles: []string{"env"}}, {SetValues: []string{"TOKEN=@file"}}} {
		_, err := BuildInstallConfig(&cobra.Command{}, versionWithOptionalSchema(), input, func(*cobra.Command, string) (string, error) { return "", errors.New("SECRET-SENTINEL") })
		require.Error(t, err)
		require.NotContains(t, err.Error(), "SECRET-SENTINEL")
	}
}

func TestConfigureEmptyMapsPreserveReadbackAndSendNoDeletions(t *testing.T) {
	current := &connectivityclient.ConnectorInstanceConfig{Env: map[string]string{"API_URL": "kept"}, Files: map[string]string{"/etc/plugin/config.yaml": "kept-file"}}
	patch := map[string]any{}
	got, err := BuildConfigureConfig(&cobra.Command{}, versionWithOptionalSchema(), current, InputOptions{ConfigFile: "patch", patch: patch}, mapReadFile(map[string]string{"patch": "env: {}\nfiles: {}\n"}))
	require.NoError(t, err)
	require.Equal(t, current, got)
	require.Empty(t, patch)
}
