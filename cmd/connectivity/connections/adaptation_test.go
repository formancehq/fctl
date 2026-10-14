package connections

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	connectivityclient "github.com/formancehq/fctl/v3/internal/connectivityclient"
)

func TestConnectionInstallWireContract(t *testing.T) {
	for _, selector := range []struct {
		name       string
		args       []string
		key, value string
	}{
		{name: "default stable", key: "channel", value: "stable"},
		{name: "channel", args: []string{"--channel=beta"}, key: "channel", value: "beta"},
		{name: "opaque pin", args: []string{"--version=build-main"}, key: "version", value: "build-main"},
	} {
		t.Run(selector.name, func(t *testing.T) {
			var body connectivityclient.ConnectorInstanceCreate
			client := mutationClientMock{
				connectorVersions: connectorVersions{getVersion: func(_ context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
					require.Equal(t, "stripe", connector)
					require.Equal(t, selector.value, version)
					return versionWithOptionalSchema(), nil
				}},
				create: func(_ context.Context, input connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
					body = input
					return &connectivityclient.ConnectorInstance{Spec: input.Spec}, nil
				},
			}
			args := append([]string{"stripe", "--ledger=main", "--set=TOKEN=private", "--set=/etc/plugin/ca.pem=private-file", "--confirm"}, selector.args...)
			_, err := executeCommand(NewInstallCommand(factoryReturning(client), nil, nil), args...)
			require.NoError(t, err)
			wire, err := json.Marshal(body)
			require.NoError(t, err)
			var decoded struct {
				Spec map[string]any `json:"spec"`
			}
			require.NoError(t, json.Unmarshal(wire, &decoded))
			require.Equal(t, map[string]any{"name": "stripe"}, decoded.Spec["connectorRef"])
			require.Equal(t, selector.value, decoded.Spec[selector.key])
			for _, obsolete := range []string{"connector", "config", "startSequence", "replicas"} {
				require.NotContains(t, decoded.Spec, obsolete)
			}
			if selector.key == "channel" {
				require.NotContains(t, decoded.Spec, "version")
			} else {
				require.NotContains(t, decoded.Spec, "channel")
			}
			require.Equal(t, map[string]any{"value": "private"}, decoded.Spec["env"].(map[string]any)["TOKEN"])
			require.Len(t, decoded.Spec["files"], 1)
		})
	}
}

func TestConnectionRejectsInvalidSelectorsBeforeClient(t *testing.T) {
	for _, name := range []string{"install", "configure"} {
		for _, flags := range [][]string{
			{"--version="}, {"--channel="}, {"--version=1.0.0", "--channel=stable"},
			{"--version=", "--channel=stable"}, {"--version=1.0.0", "--channel="}, {"--channel=unknown"},
		} {
			t.Run(name+" "+flags[0], func(t *testing.T) {
				used := false
				factory := func(*cobra.Command) (connectivityclient.Client, error) {
					used = true
					return nil, errors.New("unexpected client")
				}
				command := NewConfigureCommand(factory, nil, nil)
				args := []string{"stripe-eu", "--confirm"}
				if name == "install" {
					command = NewInstallCommand(factory, nil, nil)
					args = append(args, "--ledger=main")
				}
				_, err := executeCommand(command, append(args, flags...)...)
				require.Error(t, err)
				require.False(t, used, "invalid flags must be rejected locally")
			})
		}
	}
}

func TestConnectionConfigureLedgerIsImmutable(t *testing.T) {
	for _, ledger := range []string{"archive", ""} {
		t.Run(ledger, func(t *testing.T) {
			current := instanceFixture("stripe-eu")
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
				patch: func(context.Context, string, connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
					t.Fatal("immutable ledger must not patch")
					return nil, nil
				},
			}
			controller := NewConfigureController(factoryReturning(client), nil)
			controller.approve = func(*cobra.Command, string, ...any) bool {
				t.Fatal("immutable ledger must not ask for approval")
				return true
			}
			command := NewConfigureCommand(factoryReturning(client), nil, nil)
			require.NoError(t, command.ParseFlags([]string{"--ledger=" + ledger}))
			_, err := controller.Run(command, []string{"stripe-eu"})
			require.ErrorContains(t, err, "ledger is immutable")
		})
	}
}

func TestConnectionConfigureSelectorSwitchClearsOppositeKey(t *testing.T) {
	for _, tt := range []struct{ key, value, clear string }{
		{"version", "build-main", "channel"}, {"channel", "stable", "version"},
	} {
		t.Run(tt.key, func(t *testing.T) {
			current := instanceFixture("stripe-eu")
			if tt.key == "version" {
				current.Spec.Version = nil
				current.Spec.Channel = stringPtr("beta")
			}
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
				patch: func(_ context.Context, _ string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
					require.Equal(t, map[string]any{tt.key: tt.value, tt.clear: nil}, patch["spec"])
					return &current, nil
				},
			}
			_, err := executeCommand(NewConfigureCommand(factoryReturning(client), nil, nil), "stripe-eu", "--"+tt.key+"="+tt.value, "--confirm")
			require.NoError(t, err)
		})
	}
}

func TestConnectionChannelSelectionContract(t *testing.T) {
	for _, tt := range []struct {
		name                            string
		versions                        []string
		channel, running, want, wantErr string
	}{
		{"opaque candidates", []string{"main", "v1", "v1.2", "v1.2.3", "1.4.0", "v9.0.0-beta.1"}, "stable", "", "1.4.0", ""},
		{"opaque floor", []string{"1.2.3", "2.0.0"}, "stable", "build-main", "2.0.0", ""},
		{"strict major floor", []string{"1.2.0", "1.4.0", "2.0.0"}, "stable", "1.3.0", "1.4.0", ""},
		{"no downgrade", []string{"1.2.0", "2.0.0"}, "stable", "1.3.0", "", "no candidate"},
		{"beta admits rc", []string{"1.4.0-alpha.1", "1.3.0-beta.1", "1.3.0-rc.1"}, "beta", "", "1.3.0-rc.1", ""},
		{"exact duplicate", []string{"1.2.3", "1.2.3"}, "stable", "", "", "multiple"},
		{"duplicate excluded prerelease", []string{"1.2.3-alpha.1", "1.2.3-alpha.1", "1.2.0"}, "stable", "", "", "multiple"},
		{"opaque duplicates ignored", []string{"main", "main", "1.2.3"}, "stable", "", "1.2.3", ""},
		{"equivalent strings", []string{"1.2.3", "v1.2.3"}, "stable", "", "v1.2.3", ""},
		{"equivalent strings reversed", []string{"v1.2.3", "1.2.3"}, "stable", "", "v1.2.3", ""},
		{"opaque only", []string{"main", "v1.2"}, "stable", "", "", "no candidate"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectChannelVersion(tt.versions, tt.channel, tt.running)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestConnectionInitialCursorValidatedBeforeClient(t *testing.T) {
	for _, value := range []string{"not-base64", "YQ", "YR==", "YQ==\n", ""} {
		for _, install := range []bool{true, false} {
			used := false
			factory := func(*cobra.Command) (connectivityclient.Client, error) {
				used = true
				return nil, errors.New("unexpected client")
			}
			command := NewConfigureCommand(factory, nil, nil)
			args := []string{"stripe-eu", "--initial-cursor=" + value, "--confirm"}
			if install {
				command = NewInstallCommand(factory, nil, nil)
				args = append(args, "--ledger=main")
			}
			_, err := executeCommand(command, args...)
			require.ErrorContains(t, err, "initial-cursor")
			require.False(t, used)
		}
	}
}

func TestConnectionInitialCursorWireIsOpaque(t *testing.T) {
	client := mutationClientMock{create: func(_ context.Context, body connectivityclient.ConnectorInstanceCreate) (*connectivityclient.ConnectorInstance, error) {
		wire, err := json.Marshal(body)
		require.NoError(t, err)
		require.Contains(t, string(wire), `"initialCursor":"AP9h"`)
		require.NotContains(t, string(wire), "startSequence")
		return &connectivityclient.ConnectorInstance{Spec: body.Spec}, nil
	}}
	_, err := executeCommand(NewInstallCommand(factoryReturning(client), nil, nil), "stripe", "--ledger=main", "--initial-cursor=AP9h", "--confirm")
	require.NoError(t, err)
	current := instanceFixture("stripe-eu")
	configure := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		patch: func(_ context.Context, _ string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
			wire, err := json.Marshal(patch)
			require.NoError(t, err)
			require.JSONEq(t, `{"spec":{"initialCursor":"AP9h"}}`, string(wire))
			return &current, nil
		},
	}
	output, err := executeCommand(NewConfigureCommand(factoryReturning(configure), nil, nil), "stripe-eu", "--initial-cursor=AP9h", "--confirm")
	require.NoError(t, err)
	require.NotContains(t, output, "reset")
}

func TestConnectionConfigPatchUsesTopLevelMergeAndDeletes(t *testing.T) {
	for _, document := range []string{
		"env:\n  TOKEN: null\n  API_URL:\n    value: new-url\nfiles:\n  - path: /etc/plugin/ca.pem\n    value: new-ca\n",
		"env: null\nfiles: null\n",
	} {
		t.Run(document, func(t *testing.T) {
			// Use a wire response so this regression also exercises the new client model.
			var current connectivityclient.ConnectorInstance
			require.NoError(t, json.Unmarshal([]byte(`{"metadata":{"name":"stripe-eu"},"spec":{"connectorRef":{"name":"stripe"},"version":"1.0.0","ledger":"main","env":{"TOKEN":{"secretRef":{"name":"secrets","key":"token"}},"API_URL":{"value":"old-url"},"TIMEOUT":{"value":"kept"}},"files":[{"path":"/etc/plugin/config.yaml","value":"old-file"}]}}`), &current))
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
				connectorVersions: connectorVersions{getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
					return versionWithOptionalSchema(), nil
				}},
				patch: func(_ context.Context, _ string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
					wire, err := json.Marshal(patch)
					require.NoError(t, err)
					var decoded struct {
						Spec map[string]any `json:"spec"`
					}
					require.NoError(t, json.Unmarshal(wire, &decoded))
					require.NotContains(t, decoded.Spec, "config")
					if document == "env: null\nfiles: null\n" {
						require.Equal(t, map[string]any{"env": nil, "files": nil}, decoded.Spec)
					} else {
						env := decoded.Spec["env"].(map[string]any)
						require.Contains(t, env, "TOKEN")
						require.Nil(t, env["TOKEN"])
						require.Equal(t, map[string]any{"value": "new-url", "secretRef": nil, "configMapRef": nil}, env["API_URL"])
						files := decoded.Spec["files"].([]any)
						require.Equal(t, []any{map[string]any{"path": "/etc/plugin/ca.pem", "value": "new-ca"}}, files)
					}
					return &current, nil
				},
			}
			_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(map[string]string{"patch.yaml": document}), nil), "stripe-eu", "--config=patch.yaml", "--confirm")
			require.NoError(t, err)
		})
	}
}

func TestConnectionEnvSourceReplacementDeletesPreviousSource(t *testing.T) {
	var current connectivityclient.ConnectorInstance
	require.NoError(t, json.Unmarshal([]byte(`{"spec":{"connectorRef":{"name":"stripe"},"version":"1.0.0","ledger":"main","env":{"TOKEN":{"secretRef":{"name":"secrets","key":"token"}}}}}`), &current))
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		connectorVersions: connectorVersions{getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
			return versionWithOptionalSchema(), nil
		}},
		patch: func(_ context.Context, _ string, patch connectivityclient.ConnectorInstancePatch) (*connectivityclient.ConnectorInstance, error) {
			wire, err := json.Marshal(patch)
			require.NoError(t, err)
			var decoded struct {
				Spec map[string]any `json:"spec"`
			}
			require.NoError(t, json.Unmarshal(wire, &decoded))
			require.Equal(t, map[string]any{"value": "inline", "secretRef": nil, "configMapRef": nil}, decoded.Spec["env"].(map[string]any)["TOKEN"])
			return &current, nil
		},
	}
	_, err := executeCommand(NewConfigureCommand(factoryReturning(client), nil, nil), "stripe-eu", "--set=TOKEN=inline", "--confirm")
	require.NoError(t, err)
}

func TestConnectionConfigureCompletionUsesNewPinInsteadOfExistingChannel(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Spec.Version = nil
	current.Spec.Channel = stringPtr("beta")
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.ConnectorInstance, error) { return &current, nil },
		connectorVersions: connectorVersions{
			getVersion: func(_ context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
				require.Equal(t, "stripe", connector)
				require.Equal(t, "opaque-build", version)
				return versionWithOptionalSchema(), nil
			},
		},
	}
	command := NewConfigureCommand(factoryReturning(client), nil, nil)
	require.NoError(t, command.ParseFlags([]string{"--version=opaque-build"}))
	complete, ok := command.GetFlagCompletionFunc(setFlag)
	require.True(t, ok)
	values, directive := complete(command, []string{"stripe-eu"}, "TOKEN")
	require.Equal(t, []string{"TOKEN=\tAuthentication token"}, values)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp|cobra.ShellCompDirectiveNoSpace, directive)
}

func TestConnectionShowRendersObjectReferencesAndTopLevelConfig(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Spec.ConnectivityRef = &connectivityclient.ObjectRef{Name: "connectivity-team"}
	current.Spec.Env = map[string]connectivityclient.EnvValue{"TOKEN": {Value: stringPtr("hidden-token")}}
	output, err := executeCommand(NewShowCommand(factoryWithInstance(current)), "stripe-eu")
	require.NoError(t, err)
	require.Contains(t, output, "stripe")
	require.Contains(t, output, "connectivity-team")
	require.Contains(t, output, "TOKEN")
	require.NotContains(t, output, "hidden-token")
	require.NotContains(t, output, "Replicas")
	current.Spec.ConnectivityRef, current.Spec.ConnectorRef = nil, nil
	_, err = executeCommand(NewShowCommand(factoryWithInstance(current)), "stripe-eu")
	require.NoError(t, err)
}

func TestConnectionConfigNullAndEmptyArrayHonorPrecedenceAndRequiredKeys(t *testing.T) {
	current := &connectivityclient.ConnectorInstanceConfig{
		Env:   map[string]connectivityclient.EnvValue{"TOKEN": {Value: stringPtr("old-token")}},
		Files: []connectivityclient.FileMount{{Path: "/etc/plugin/ca.pem", Value: stringPtr("old-ca")}},
	}
	desired, err := BuildConfigureConfig(&cobra.Command{}, versionWithOptionalSchema(), current,
		InputOptions{ConfigFile: "patch", SetValues: []string{"TOKEN=new-token"}},
		mockReadFile(map[string]string{"patch": "env: null\nfiles: []\n"}))
	require.NoError(t, err)
	require.Equal(t, "new-token", *desired.Env["TOKEN"].Value)
	require.NotNil(t, desired.Files)
	require.Empty(t, desired.Files)
	patch := configurationPatch(current, desired)
	wire, err := json.Marshal(patch)
	require.NoError(t, err)
	require.Contains(t, string(wire), `"files":[]`)
	require.Equal(t, "old-token", *current.Env["TOKEN"].Value, "builder must preserve the fetched instance")
	require.Len(t, current.Files, 1)
	_, err = BuildConfigureConfig(&cobra.Command{}, versionWithFullSchema(), current,
		InputOptions{ConfigFile: "patch"}, mockReadFile(map[string]string{"patch": "env: null\nfiles: null\n"}))
	require.ErrorContains(t, err, "missing required configuration keys")
}
