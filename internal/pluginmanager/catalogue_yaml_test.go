package pluginmanager

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func TestCatalogueYAMLAndJSONPreserveManifest(t *testing.T) {
	m := manager(t)
	release := Release{
		Service: "ledger", ServiceVersion: "3.0.0-beta.10", Revision: 2,
		Platform: CurrentPlatform(), SHA256: strings.Repeat("a", 64),
		Artifact: Artifact{Registry: "https://ghcr.io", Repository: "formancehq/ledger-fctl-plugin", Digest: "sha256:" + strings.Repeat("b", 64)},
		Manifest: manifest("3.0.0-beta.10"),
	}
	release.Manifest.Root.Flags = []pluginsdk.FlagSpec{{Name: "ledger", Type: "string", Default: "90071992547409930001", Persistent: true}}
	release.Manifest.Root.Subcommands[0].Inputs = []pluginsdk.InputSpec{{
		Title: "Ledger", Kind: "select", Flag: "ledger", Required: true,
		Source: &pluginsdk.ChoiceSource{CommandPath: []string{"ledger", "list"}, ValueField: "name", MatchFields: map[string][]string{"status": {"yes", "no"}}},
	}}
	catalogue := Catalogue{SchemaVersion: SchemaVersion, Releases: []Release{release}}
	data, err := yaml.Marshal(catalogue)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "registry.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{path, catalogueFile(t, release)} {
		got, err := m.Resolve(t.Context(), source, "ledger", release.ServiceVersion, 0)
		if err != nil || !reflect.DeepEqual(got, release) {
			t.Fatalf("manifest changed after catalogue decoding: %+v, %v", got, err)
		}
		if _, err := m.Resolve(t.Context(), source, "ledger", "3.0.0-beta.11", 0); !errors.Is(err, ErrNoRelease) {
			t.Fatalf("selected a different service version: %v", err)
		}
	}
}

func TestCatalogueYAMLDiscoveryValidation(t *testing.T) {
	for _, test := range []struct {
		name, data string
		valid      bool
	}{
		{"empty", "# Official registry\nschemaVersion: 1\nreleases: []\n", true},
		{"unknown-field", "schemaVersion: 1\nreleases: []\nrelases: []\n", false},
		{"duplicate-field", "schemaVersion: 1\nschemaVersion: 2\nreleases: []\n", false},
		{"schema", "schemaVersion: 2\nreleases: []\n", false},
		{"invalid-release", "schemaVersion: 1\nreleases:\n  - service: ledger\n    revision: 1\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := manager(t)
			m.client.Transport = catalogueTransport(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != DefaultCatalogue || request.Header.Get("Authorization") != "" {
					t.Fatalf("unexpected discovery request: %s", request.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(test.data)), Header: make(http.Header)}, nil
			})
			catalogue, err := m.Discover(t.Context(), DefaultCatalogue)
			if (err == nil) != test.valid {
				t.Fatalf("discovery: %+v %v", catalogue, err)
			}
		})
	}
}

type catalogueTransport func(*http.Request) (*http.Response, error)

func (f catalogueTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
