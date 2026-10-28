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
			var body connectivityclient.ConnectionCreate
			client := mutationClientMock{
				connectorVersions: connectorVersions{getVersion: func(_ context.Context, connector, version string) (*connectivityclient.ConnectorVersion, error) {
					require.Equal(t, "stripe", connector)
					require.Equal(t, selector.value, version)
					return versionWithOptionalSchema(), nil
				}},
				create: func(_ context.Context, input connectivityclient.ConnectionCreate) (*connectivityclient.Connection, error) {
					body = input
					return &connectivityclient.Connection{Configuration: input.Configuration}, nil
				},
			}
			args := append([]string{"stripe", "--ledger=main", "--set=TOKEN=private", "--set=/etc/plugin/ca.pem=private-file", "--confirm"}, selector.args...)
			_, err := executeCommand(NewInstallCommand(factoryReturning(client), nil, nil), args...)
			require.NoError(t, err)
			wire, err := json.Marshal(body)
			require.NoError(t, err)
			var decoded struct {
				Configuration map[string]any `json:"configuration"`
			}
			require.NoError(t, json.Unmarshal(wire, &decoded))
			require.Equal(t, "stripe", decoded.Configuration["connector"])
			require.Equal(t, selector.value, decoded.Configuration[selector.key])
			for _, obsolete := range []string{"connectorRef", "config", "startSequence", "replicas"} {
				require.NotContains(t, decoded.Configuration, obsolete)
			}
			if selector.key == "channel" {
				require.NotContains(t, decoded.Configuration, "version")
			} else {
				require.NotContains(t, decoded.Configuration, "channel")
			}
			require.Equal(t, "private", decoded.Configuration["env"].(map[string]any)["TOKEN"])
			require.Len(t, decoded.Configuration["files"], 1)
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
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				patch: func(context.Context, string, connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
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
				current.Configuration.Version = nil
				current.Configuration.Channel = stringPtr("beta")
			}
			client := configureClientMock{
				getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
				patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
					require.Equal(t, map[string]any{tt.key: tt.value, tt.clear: nil}, patch["configuration"])
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
	client := mutationClientMock{create: func(_ context.Context, body connectivityclient.ConnectionCreate) (*connectivityclient.Connection, error) {
		wire, err := json.Marshal(body)
		require.NoError(t, err)
		require.Contains(t, string(wire), `"initialCursor":"AP9h"`)
		require.NotContains(t, string(wire), "startSequence")
		return &connectivityclient.Connection{Configuration: body.Configuration}, nil
	}}
	_, err := executeCommand(NewInstallCommand(factoryReturning(client), nil, nil), "stripe", "--ledger=main", "--initial-cursor=AP9h", "--confirm")
	require.NoError(t, err)
	current := instanceFixture("stripe-eu")
	configure := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
		patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
			wire, err := json.Marshal(patch)
			require.NoError(t, err)
			require.JSONEq(t, `{"configuration":{"initialCursor":"AP9h"}}`, string(wire))
			return &current, nil
		},
	}
	output, err := executeCommand(NewConfigureCommand(factoryReturning(configure), nil, nil), "stripe-eu", "--initial-cursor=AP9h", "--confirm")
	require.NoError(t, err)
	require.NotContains(t, output, "reset")
}

func TestConnectionConfigPatchCarriesIntentForRedactedValues(t *testing.T) {
	for _, document := range []string{
		"env:\n  TOKEN: null\n  API_URL: new-url\nfiles:\n  /etc/plugin/ca.pem: null\n",
		`{"env":{"TOKEN":null,"API_URL":"new-url"},"files":{"/etc/plugin/ca.pem":null}}`,
		"env: null\nfiles: null\n",
	} {
		t.Run(document, func(t *testing.T) {
			current := instanceFixture("stripe-eu")
			// TOKEN and the password file are intentionally absent from the readback.
			current.Configuration.Env = map[string]string{"API_URL": "old-url", "TIMEOUT": "kept"}
			current.Configuration.Files = map[string]string{"/etc/plugin/config.yaml": "kept-file"}
			var got connectivityclient.ConnectionPatch
			client := configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil }, connectorVersions: connectorVersions{getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
				return versionWithFullSchema(), nil
			}}, patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
				got = patch
				return &current, nil
			}}
			_, err := executeCommand(NewConfigureCommand(factoryReturning(client), mockReadFile(map[string]string{"patch": document}), nil), "stripe-eu", "--config=patch", "--confirm")
			require.NoError(t, err)
			configuration := got["configuration"].(map[string]any)
			if document == "env: null\nfiles: null\n" {
				require.Equal(t, map[string]any{"env": nil, "files": nil}, configuration)
			} else {
				require.Equal(t, map[string]any{"TOKEN": nil, "API_URL": "new-url"}, configuration["env"])
				require.Equal(t, map[string]any{"/etc/plugin/ca.pem": nil}, configuration["files"])
			}
		})
	}
}

func TestConnectionConfigurePreservesOmittedRequiredPasswords(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Configuration.Env = map[string]string{"API_URL": "old-url"}
	current.Configuration.Files = map[string]string{"/etc/plugin/config.yaml": "kept-file"}
	var got connectivityclient.ConnectionPatch
	client := configureClientMock{getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil }, connectorVersions: connectorVersions{getVersion: func(context.Context, string, string) (*connectivityclient.ConnectorVersion, error) {
		return versionWithFullSchema(), nil
	}}, patch: func(_ context.Context, _ string, patch connectivityclient.ConnectionPatch) (*connectivityclient.Connection, error) {
		got = patch
		return &current, nil
	}}
	_, err := executeCommand(NewConfigureCommand(factoryReturning(client), nil, nil), "stripe-eu", "--set=API_URL=new-url", "--confirm")
	require.NoError(t, err)
	require.Equal(t, connectivityclient.ConnectionPatch{"configuration": map[string]any{"env": map[string]any{"API_URL": "new-url"}}}, got)
}

func TestConnectionConfigureCompletionUsesNewPinInsteadOfExistingChannel(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Configuration.Version = nil
	current.Configuration.Channel = stringPtr("beta")
	client := configureClientMock{
		getInstance: func(context.Context, string) (*connectivityclient.Connection, error) { return &current, nil },
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

func TestConnectionShowAllowsUnboundPublicConfiguration(t *testing.T) {
	current := instanceFixture("stripe-eu")
	current.Configuration.Connector = ""
	current.Configuration.Env = map[string]string{"TOKEN": "hidden-token"}
	output, err := executeCommand(NewShowCommand(factoryWithInstance(current)), "stripe-eu")
	require.NoError(t, err)
	require.Contains(t, output, "TOKEN")
	require.NotContains(t, output, "hidden-token")
}

func TestConnectionConfigNullEntriesHonorSetPrecedence(t *testing.T) {
	current := &connectivityclient.ConnectionConfig{Env: map[string]string{"TOKEN": "old-token"}, Files: map[string]string{"/etc/plugin/ca.pem": "old-ca"}}
	patch := map[string]any{}
	desired, err := BuildConfigureConfig(&cobra.Command{}, versionWithFullSchema(), current, InputOptions{patch: patch, ConfigFile: "patch", SetValues: []string{"TOKEN=new-token"}}, mockReadFile(map[string]string{"patch": "env:\n  TOKEN: null\nfiles:\n  /etc/plugin/ca.pem: null\n"}))
	require.NoError(t, err)
	require.Equal(t, "new-token", desired.Env["TOKEN"])
	require.Equal(t, map[string]any{"env": map[string]any{"TOKEN": "new-token"}, "files": map[string]any{"/etc/plugin/ca.pem": nil}}, patch)
	require.Equal(t, "old-token", current.Env["TOKEN"])
	require.Equal(t, "old-ca", current.Files["/etc/plugin/ca.pem"])
}
