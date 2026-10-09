package cmd

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func discoveryPreparation(t *testing.T, transport http.RoundTripper) ledgerPreparation {
	t.Helper()
	root := &cobra.Command{Use: "fctl"}
	settings := &connection.Settings{}
	settings.Bind(root)
	settings.Directory = t.TempDir()
	settings.Options.AuthMode = "none"
	settings.Options.LedgerURL = "http://127.0.0.1:1"
	manager, err := pluginmanager.New(t.TempDir(), &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	return ledgerPreparation{root: root, settings: settings, manager: manager}
}

func TestOfficialDiscoveryRequiresPublishedLedgerPlugin(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	for _, test := range []struct {
		name, data string
		status     int
		want       error
	}{
		{"empty", "schemaVersion: 1\nreleases: []\n", http.StatusOK, pluginmanager.ErrNoRelease},
		{"unavailable", "", http.StatusServiceUnavailable, pluginmanager.ErrCatalogueUnavailable},
		{"invalid", "schemaVersion: 1\nreleases: invalid\n", http.StatusOK, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			prep := discoveryPreparation(t, discoveryTransport(func(request *http.Request) (*http.Response, error) {
				requests++
				if request.URL.String() != pluginmanager.DefaultCatalogue {
					t.Fatalf("unexpected registry: %s", request.URL)
				}
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.data)), Header: make(http.Header)}, nil
			}))
			_, err := prep.resolve(t.Context(), pluginBootstrap{commands: []string{"ledger", "list"}})
			if err == nil || errors.Is(err, pluginmanager.ErrNotInstalled) || (test.want != nil && !errors.Is(err, test.want)) || requests != 1 {
				t.Fatalf("want=%v, requests=%d, error=%v", test.want, requests, err)
			}
		})
	}
}

func TestOfficialDiscoverySkipsOfflineMetadataAndOtherModules(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	prep := discoveryPreparation(t, discoveryTransport(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("metadata attempted discovery: %s", request.URL)
		return nil, context.Canceled
	}))
	for _, plan := range []pluginBootstrap{
		{commands: []string{"ledger"}, help: true},
		{commands: []string{"__complete", "ledger"}},
		{commands: []string{"auth", "info"}},
		{},
	} {
		if _, err := prep.resolve(t.Context(), plan); !errors.Is(err, pluginmanager.ErrNotInstalled) {
			t.Fatalf("offline bootstrap: %v", err)
		}
	}
}

func TestPluginCatalogueUsesOfficialDefaultAndExplicitOverrides(t *testing.T) {
	t.Setenv("FCTL_PLUGIN_CATALOGUE", "")
	prep := discoveryPreparation(t, discoveryTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("source selection attempted a download")
		return nil, context.Canceled
	}))
	for _, test := range []struct{ explicit, environment, want string }{
		{"", "", pluginmanager.DefaultCatalogue},
		{"", "https://example.com/environment.yaml", "https://example.com/environment.yaml"},
		{"explicit.yaml", "https://example.com/environment.yaml", "explicit.yaml"},
	} {
		t.Setenv("FCTL_PLUGIN_CATALOGUE", test.environment)
		got, err := pluginCatalogue(prep.root, prep.settings, prep.manager, test.explicit)
		if err != nil || got != test.want {
			t.Fatalf("catalogue: %q, %v", got, err)
		}
	}
}

type discoveryTransport func(*http.Request) (*http.Response, error)

func (f discoveryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
