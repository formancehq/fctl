package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/formancehq/fctl/v3/cmd/connectivity/connections/configuration"
	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

var templateNow = time.Date(2026, time.November, 10, 9, 0, 0, 0, time.UTC)

func templateSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"env": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"SECRET_KEY", "ENVIRONMENT"},
			"properties": map[string]any{
				"SECRET_KEY":  map[string]any{"type": "string", "format": "password", "description": "Stripe secret API key used to authenticate requests."},
				"ENVIRONMENT": map[string]any{"type": "string", "enum": []any{"sandbox", "live"}, "description": "Stripe environment the key belongs to."},
				"PAGE_SIZE":   map[string]any{"type": "string", "default": "100", "description": "Number of objects fetched per page."},
				"API_URL":     map[string]any{"type": "string", "format": "uri", "description": "Override the Stripe API base URL."},
				"DEBUG":       map[string]any{"type": "string", "default": "false"},
				"REGION":      map[string]any{"$ref": "#/$defs/region"},
			},
			"$defs": map[string]any{
				"region": map[string]any{"type": "string", "enum": []any{"eu", "us"}, "default": "eu", "description": "Data residency region."},
			},
		},
		"files": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"/etc/plugin/client.key"},
			"properties": map[string]any{
				"/etc/plugin/client.key": map[string]any{"type": "string", "format": "password", "description": "Private key used for mTLS with the provider."},
			},
		},
		"presentation": map[string]any{
			"groups": []any{
				map[string]any{"title": "Credentials", "properties": []any{"env.SECRET_KEY", "env.ENVIRONMENT", "files/etc/plugin/client.key"}},
				map[string]any{"title": "Ingestion", "properties": []any{"env.PAGE_SIZE", "env.REGION"}},
				map[string]any{"title": "Tuning", "advanced": true, "properties": []any{"env.API_URL"}},
			},
		},
		"requirements": []any{
			map[string]any{"description": "Live keys require ENVIRONMENT=live."},
		},
	}
}

func templateClient(t *testing.T, schema map[string]any, gotVersion *string) *formance.FormanceV1 {
	t.Helper()
	connector := connectorFixture("stripe")
	return sdkFor(connectorTransportMock{
		get: func(_ context.Context, name string) (*connectivity.Connector, error) {
			require.Equal(t, "stripe", name)
			return &connector, nil
		},
		getVersion: func(_ context.Context, _, version string) (*connectivity.ConnectorVersion, error) {
			if gotVersion != nil {
				*gotVersion = version
			}
			resolved := version
			switch version {
			case "stable":
				resolved = "2.0.0"
			case "beta":
				resolved = "2.1.0-beta.1"
			}
			return &connectivity.ConnectorVersion{Version: resolved, ConfigSchema: schema}, nil
		},
	})
}

func testNewTemplateCommand(client *formance.FormanceV1, ignored func(context.Context, string) (bool, bool)) *cobra.Command {
	controller := NewTemplateController()
	controller.now = func() time.Time { return templateNow }
	if ignored == nil {
		ignored = func(context.Context, string) (bool, bool) { return false, false }
	}
	controller.gitIgnored = ignored
	command := NewTemplateCommand()
	command.RunE = func(cmd *cobra.Command, args []string) error {
		renderable, err := controller.run(cmd, args, client)
		if err != nil {
			return err
		}
		return fctl.WithRender(cmd, args, controller, renderable)
	}
	return command
}

func TestTemplateRendersCommentedYAML(t *testing.T) {
	var gotVersion string
	output, err := executeCommand(testNewTemplateCommand(templateClient(t, templateSchema(), &gotVersion), nil), "stripe", "--file", "-")
	require.NoError(t, err)
	require.Equal(t, "stable", gotVersion, "defaults to the stable channel, like connections install")

	expected := `# Connection configuration for connector "stripe"
# Generated from version 2.0.0 (channel: stable) on 2026-11-10.
#
# Fill in the values, then install with:
#   fctl connectivity connections install stripe \
#     --ledger <ledger> --channel stable --config stripe.yaml
#
# Keys left empty are unset: install fails while a required key is unset.
# Values are strings: quote numbers and booleans, for example "100" or "true".
#
# Rules:
#   • Live keys require ENVIRONMENT=live.
#
# ⚠ This file contains credentials once filled in. To keep them safe:
#   • Do not commit it: add it to your .gitignore.
#   • Keep it readable only by you: chmod 600 stripe.yaml (fctl creates it this way).
#   • Delete it once the connection is installed, or store it in a password manager.
#   • Prefer sandbox/test credentials while trying things out.
#   • To keep secrets out of this file entirely, leave them empty here and pass them
#     at install time instead: --set KEY=@./path/to/secret
#   • If this file leaks, rotate the credentials with the provider right away.

env:

  # ── Credentials ─────────────────────────────────────────────

  # Stripe secret API key used to authenticate requests.
  # Required · secret
  SECRET_KEY:

  # Stripe environment the key belongs to.
  # Required · string · one of: sandbox, live
  ENVIRONMENT:

  # ── Ingestion ───────────────────────────────────────────────

  # Number of objects fetched per page.
  # Optional · string · default: 100
  # PAGE_SIZE: "100"

  # Data residency region.
  # Optional · string · one of: eu, us · default: eu
  # REGION: "eu"

  # ── Tuning (advanced) ───────────────────────────────────────

  # Override the Stripe API base URL.
  # Optional · uri
  # API_URL:

  # ── Other ───────────────────────────────────────────────────

  # Optional · string · default: false
  # DEBUG: "false"

files:

  # ── Credentials ─────────────────────────────────────────────

  # Private key used for mTLS with the provider.
  # Required · secret · file content, mounted at this path in the connector
  /etc/plugin/client.key:
`
	require.Equal(t, expected, output)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(output), &parsed), "template must be valid YAML")
}

func readFrom(contents string) configuration.ReadFileFunc {
	return func(*cobra.Command, string) (string, error) { return contents, nil }
}

func TestTemplateRoundTripsThroughInstallConfig(t *testing.T) {
	schema := templateSchema()
	output, err := executeCommand(testNewTemplateCommand(templateClient(t, schema, nil), nil), "stripe", "--file", "-")
	require.NoError(t, err)
	version := &connectivity.ConnectorVersion{Version: "2.0.0", ConfigSchema: schema}
	inputs := configuration.InputOptions{ConfigFile: "stripe.yaml"}

	// Unfilled: only the required keys are reported, nothing is unknown.
	_, err = configuration.BuildInstallConfig(&cobra.Command{}, version, inputs, readFrom(output))
	require.EqualError(t, err, "missing required configuration keys: /etc/plugin/client.key, ENVIRONMENT, SECRET_KEY")

	// Filled in and with an optional default uncommented.
	filled := strings.NewReplacer(
		"  SECRET_KEY:\n", "  SECRET_KEY: sk_test_123\n",
		"  ENVIRONMENT:\n", "  ENVIRONMENT: sandbox\n",
		"  /etc/plugin/client.key:\n", "  /etc/plugin/client.key: |\n    -----BEGIN PRIVATE KEY-----\n",
		`  # PAGE_SIZE: "100"`, `  PAGE_SIZE: "100"`,
	).Replace(output)
	config, err := configuration.BuildInstallConfig(&cobra.Command{}, version, inputs, readFrom(filled))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"SECRET_KEY": "sk_test_123", "ENVIRONMENT": "sandbox", "PAGE_SIZE": "100"}, config.Env)
	require.Equal(t, map[string]string{"/etc/plugin/client.key": "-----BEGIN PRIVATE KEY-----\n"}, config.Files)
}

func TestTemplateUsesChannel(t *testing.T) {
	var gotVersion string
	output, err := executeCommand(testNewTemplateCommand(templateClient(t, templateSchema(), &gotVersion), nil), "stripe", "--channel", "beta", "--file", "-")
	require.NoError(t, err)
	require.Equal(t, "beta", gotVersion)
	require.Contains(t, output, "# Generated from version 2.1.0-beta.1 (channel: beta) on 2026-11-10.")
	require.Contains(t, output, "--ledger <ledger> --channel beta --config stripe.yaml")
}

func TestTemplateRejectsVersionAndChannel(t *testing.T) {
	_, err := executeCommand(testNewTemplateCommand(templateClient(t, templateSchema(), nil), nil), "stripe", "--version", "2.0.0", "--channel", "beta", "--file", "-")
	require.EqualError(t, err, "--version and --channel cannot be combined")
}

func TestTemplateWithoutSecretsOmitsSecurityTips(t *testing.T) {
	schema := map[string]any{
		"env": map[string]any{
			"type":       "object",
			"properties": map[string]any{"API_URL": map[string]any{"type": "string"}},
		},
	}
	output, err := executeCommand(testNewTemplateCommand(templateClient(t, schema, nil), nil), "stripe", "--file", "-")
	require.NoError(t, err)
	require.NotContains(t, output, "credentials")
	require.NotContains(t, output, "files:")
	require.Contains(t, output, "env:\n\n  # Optional · string\n  # API_URL:\n")
}

func TestTemplateWithoutFields(t *testing.T) {
	output, err := executeCommand(testNewTemplateCommand(templateClient(t, nil, nil), nil), "stripe", "--file", "-")
	require.NoError(t, err)
	require.Contains(t, output, "# This connector declares no configuration fields.\n")
	require.NotContains(t, output, "env:")
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(output), &parsed))
	require.Empty(t, parsed)
}

func emptyChannelClient(t *testing.T, gotVersion *string) *formance.FormanceV1 {
	t.Helper()
	connector := connectorFixture("stripe")
	return sdkFor(connectorTransportMock{
		get: func(context.Context, string) (*connectivity.Connector, error) { return &connector, nil },
		getVersion: func(_ context.Context, _, version string) (*connectivity.ConnectorVersion, error) {
			*gotVersion = version
			return nil, &sdkerrors.SDKError{StatusCode: http.StatusNotFound, Body: `{"errorCode":"channel_empty","errorMessage":"No validated version"}`}
		},
	})
}

func TestTemplateWithoutStableReleaseAsksForAnExplicitSelector(t *testing.T) {
	var gotVersion string
	_, err := executeCommand(testNewTemplateCommand(emptyChannelClient(t, &gotVersion), nil), "stripe", "--file", "-")
	require.Equal(t, "stable", gotVersion)
	require.ErrorContains(t, err, `connector "stripe" has no release on the stable channel; set --channel or --version explicitly`)
	var apiErr *connectivity.Error
	require.ErrorAs(t, err, &apiErr, "the native API error stays available")
	require.Equal(t, "channel_empty", *apiErr.ErrorCode)
}

func TestTemplateWithEmptyExplicitChannelAsksForAnExplicitSelector(t *testing.T) {
	var gotVersion string
	_, err := executeCommand(testNewTemplateCommand(emptyChannelClient(t, &gotVersion), nil), "stripe", "--channel", "rc", "--file", "-")
	require.Equal(t, "rc", gotVersion)
	require.ErrorContains(t, err, `connector "stripe" has no release on the rc channel; set --channel or --version explicitly`)
}

func TestTemplateWithMissingPinnedVersionKeepsTheAPIError(t *testing.T) {
	var gotVersion string
	_, err := executeCommand(testNewTemplateCommand(emptyChannelClient(t, &gotVersion), nil), "stripe", "--version", "9.9.9", "--file", "-")
	require.Equal(t, "9.9.9", gotVersion)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "set --channel or --version")
}

func TestTemplateWritesPrivateFileAndRefusesOverwrite(t *testing.T) {
	t.Chdir(t.TempDir())
	client := templateClient(t, templateSchema(), nil)

	output, err := executeCommand(testNewTemplateCommand(client, nil), "stripe")
	require.NoError(t, err)
	require.Contains(t, output, `Configuration template for "stripe" (2.0.0) written to stripe.yaml.`)
	info, err := os.Stat("stripe.yaml")
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	content, err := os.ReadFile("stripe.yaml")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(content), `# Connection configuration for connector "stripe"`))

	require.NoError(t, os.WriteFile("stripe.yaml", []byte("filled: in\n"), 0o644)) //nolint:gosec // test fixture
	_, err = executeCommand(testNewTemplateCommand(client, nil), "stripe")
	require.EqualError(t, err, "stripe.yaml already exists; use --force to overwrite it")
	content, err = os.ReadFile("stripe.yaml")
	require.NoError(t, err)
	require.Equal(t, "filled: in\n", string(content), "an existing file is never touched without --force")

	_, err = executeCommand(testNewTemplateCommand(client, nil), "stripe", "--force")
	require.NoError(t, err)
	info, err = os.Stat("stripe.yaml")
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "--force restores private permissions")
}

func TestTemplateWarnsWhenFileIsNotGitIgnored(t *testing.T) {
	dir := t.TempDir()
	client := templateClient(t, templateSchema(), nil)
	notIgnored := func(context.Context, string) (bool, bool) { return false, true }
	ignored := func(context.Context, string) (bool, bool) { return true, true }

	output, err := executeCommand(testNewTemplateCommand(client, notIgnored), "stripe", "--file", filepath.Join(dir, "a.yaml"))
	require.NoError(t, err)
	require.Contains(t, output, "Warning: "+filepath.Join(dir, "a.yaml")+" is inside a git repository and not ignored.")

	output, err = executeCommand(testNewTemplateCommand(client, ignored), "stripe", "--file", filepath.Join(dir, "b.yaml"))
	require.NoError(t, err)
	require.NotContains(t, output, "Warning")
}

func TestGitCheckIgnore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	ctx := context.Background()
	outside := t.TempDir()
	_, known := gitCheckIgnore(ctx, filepath.Join(outside, "stripe.yaml"))
	require.False(t, known, "outside a work tree the answer is unknown")

	repo := t.TempDir()
	init := exec.Command("git", "init", "-q", repo) // #nosec G204 -- test temp directory
	require.NoError(t, init.Run())
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("*.secret.yaml\n"), 0o600))

	ignored, known := gitCheckIgnore(ctx, filepath.Join(repo, "stripe.yaml"))
	require.True(t, known)
	require.False(t, ignored)
	ignored, known = gitCheckIgnore(ctx, filepath.Join(repo, "stripe.secret.yaml"))
	require.True(t, known)
	require.True(t, ignored)
}

func TestTemplateStoreJSON(t *testing.T) {
	controller := NewTemplateController()
	controller.now = func() time.Time { return templateNow }
	command := NewTemplateCommand()
	command.SetArgs([]string{"stripe", "--file", "-"})
	command.SetContext(context.Background())
	require.NoError(t, command.ParseFlags([]string{"--file", "-"}))
	_, err := controller.run(command, []string{"stripe"}, templateClient(t, templateSchema(), nil))
	require.NoError(t, err)
	encoded, err := json.Marshal(controller.GetStore())
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, "stripe", decoded["connector"])
	require.Equal(t, "2.0.0", decoded["version"])
	require.NotContains(t, decoded, "file")
	require.Contains(t, decoded["content"], "SECRET_KEY:")
}
