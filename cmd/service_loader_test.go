package cmd_test

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/v4/cmd"
	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func loaderManifest(service string) pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: service, Service: service, Version: "1.0.0", ProtocolVersion: pluginsdk.ProtocolVersion,
		Root: pluginsdk.CommandSpec{Use: service, Short: "Cached external " + service, Subcommands: []pluginsdk.CommandSpec{{Use: "probe", Short: "External probe", Runnable: true}}}}
}

func loaderSettings(t *testing.T) (*cobra.Command, *connection.Settings) {
	t.Helper()
	root := &cobra.Command{Use: "fctl"}
	settings := &connection.Settings{}
	settings.Bind(root)
	settings.Directory = t.TempDir()
	loaderFlag(t, root, "auth-mode", "none")
	return root, settings
}

func loaderFlag(t *testing.T, root *cobra.Command, name, value string) {
	t.Helper()
	if err := root.PersistentFlags().Set(name, value); err != nil {
		t.Fatal(err)
	}
}

func installLoaderFixture(t *testing.T, manager *pluginmanager.Manager, target pluginmanager.Target, service string) pluginmanager.Lock {
	t.Helper()
	path := filepath.Join(t.TempDir(), "not-an-executable")
	if err := os.WriteFile(path, []byte("cached help must never start this file"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := manager.InstallLocal(t.Context(), path, target, loaderManifest(service))
	if err != nil {
		t.Fatal(err)
	}
	return lock
}

type discoveryTransport func(*http.Request) (*http.Response, error)

func (f discoveryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

//nolint:gocognit // One offline lifecycle verifies both cached providers.
func TestServiceLoaderOfflineRoots(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "https://invalid.example/catalogue")
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "auth-mode", "cloud")
	loaderFlag(t, root, "issuer", "https://issuer.example")
	manager, err := pluginmanager.New(filepath.Join(settings.Directory, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	target := pluginmanager.Target{Profile: "", Organization: "org", Stack: "stack", Endpoint: settings.Options.Issuer}
	for _, service := range []string{"auth", "ledger"} {
		installLoaderFixture(t, manager, target, service)
	}
	for _, service := range []string{"auth", "ledger"} {
		for _, suffix := range [][]string{{service, "--help"}, {"__complete", service, ""}} {
			args := append([]string{"--config-dir", settings.Directory, "--auth-mode", "cloud", "--issuer", settings.Options.Issuer}, suffix...)
			command := cmd.NewRootCommandWithArgs(t.Context(), args)
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)
			command.SetArgs(args)
			if err := command.ExecuteContext(t.Context()); err != nil {
				t.Fatalf("offline %v: %v", suffix, err)
			}
			if !strings.Contains(output.String(), "probe") {
				t.Fatalf("cached metadata not registered: %s", output.String())
			}
			count := 0
			for _, child := range command.Commands() {
				if child.Name() == service {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("%s roots: %d", service, count)
			}
		}
	}
}

func TestAuthLoaderNilArgsUsesCachedMetadataOrPlaceholder(t *testing.T) {
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "auth-url", "https://auth.example")
	t.Setenv("FCTL_CONFIG_DIR", settings.Directory)
	t.Setenv("FCTL_AUTH_MODE", "none")
	t.Setenv("FCTL_AUTH_URL", "https://auth.example")
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "https://invalid.example/catalogue")
	base := http.DefaultTransport
	http.DefaultTransport = discoveryTransport(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("nil-args Auth metadata accessed network: %s", request.URL)
		return nil, context.Canceled
	})
	t.Cleanup(func() { http.DefaultTransport = base })

	command := cmd.NewRootCommand()
	auth, _, err := command.Find([]string{"auth"})
	if err != nil || auth == command || auth.HasSubCommands() {
		t.Fatalf("uncached Auth must expose only a placeholder: %v", err)
	}
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"auth", "--help"})
	if err := command.ExecuteContext(t.Context()); err != nil || !strings.Contains(output.String(), "plugins sync --service auth") {
		t.Fatalf("placeholder help: %s %v", output.String(), err)
	}

	manager, err := pluginmanager.New(filepath.Join(settings.Directory, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	installLoaderFixture(t, manager, pluginmanager.Target{Endpoint: "https://auth.example"}, "auth")
	for _, args := range [][]string{{"auth", "--help"}, {"__complete", "auth", ""}} {
		command = cmd.NewRootCommand()
		output.Reset()
		command.SetOut(&output)
		command.SetErr(&output)
		command.SetArgs(args)
		if err := command.ExecuteContext(t.Context()); err != nil || !strings.Contains(output.String(), "probe") {
			t.Fatalf("nil-args cached Auth %v: %s %v", args, output.String(), err)
		}
	}
}

func TestLedgerLoaderNilArgsUsesCachedMetadataOrPlaceholder(t *testing.T) {
	root, settings := loaderSettings(t)
	loaderFlag(t, root, "ledger-url", "https://ledger.example")
	t.Setenv("FCTL_CONFIG_DIR", settings.Directory)
	t.Setenv("FCTL_AUTH_MODE", "none")
	t.Setenv("FCTL_LEDGER_URL", "https://ledger.example")
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "https://invalid.example/catalogue")
	base := http.DefaultTransport
	http.DefaultTransport = discoveryTransport(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("nil-args Ledger metadata accessed network: %s", request.URL)
		return nil, context.Canceled
	})
	t.Cleanup(func() { http.DefaultTransport = base })

	command := cmd.NewRootCommand()
	ledger, _, err := command.Find([]string{"ledger"})
	if err != nil || ledger == command || ledger.HasSubCommands() {
		t.Fatalf("uncached Ledger must expose only a placeholder: %v", err)
	}
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"ledger", "--help"})
	if err := command.ExecuteContext(t.Context()); err != nil || !strings.Contains(output.String(), "plugins sync --service ledger") {
		t.Fatalf("placeholder help: %s %v", output.String(), err)
	}

	manager, err := pluginmanager.New(filepath.Join(settings.Directory, "plugins"), nil)
	if err != nil {
		t.Fatal(err)
	}
	installLoaderFixture(t, manager, pluginmanager.Target{Endpoint: "https://ledger.example"}, "ledger")
	for _, args := range [][]string{{"ledger", "--help"}, {"__complete", "ledger", ""}} {
		command = cmd.NewRootCommand()
		output.Reset()
		command.SetOut(&output)
		command.SetErr(&output)
		command.SetArgs(args)
		if err := command.ExecuteContext(t.Context()); err != nil || !strings.Contains(output.String(), "probe") {
			t.Fatalf("nil-args cached Ledger %v: %s %v", args, output.String(), err)
		}
	}
}
