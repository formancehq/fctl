package pluginmanager

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func referenceIndex(version, address string, data []byte) Catalogue {
	return Catalogue{SchemaVersion: RegistrySchemaVersion, Plugins: map[string]ProductCatalogues{
		"ledger": {Releases: []CatalogueReference{{ServiceVersion: version, Catalogue: address, SHA256: checksum(data)}}},
	}}
}

func jsonFixture(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestReferencedCatalogueSyncSelectsVersionAndRetainsOfflineLock(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	first := publish(t, m, registry, "3.0.0", 1)
	latest := publish(t, m, registry, "3.0.0", 2)
	const source = "https://catalogue.example/registry.yaml"
	const productURL = "https://catalogue.example/ledger-3.0.0.json"
	product := jsonFixture(t, Catalogue{SchemaVersion: SchemaVersion, Releases: []Release{first, latest}})
	index := referenceIndex("3.0.0", productURL, product)
	index.Plugins["auth"] = ProductCatalogues{Releases: []CatalogueReference{{
		ServiceVersion: "2.5.2", Catalogue: "https://catalogue.example/never-auth.json", SHA256: strings.Repeat("a", 64),
	}}}
	index.Plugins["ledger"] = ProductCatalogues{Releases: append(index.Plugins["ledger"].Releases, CatalogueReference{
		ServiceVersion: "3.0.1", Catalogue: "https://catalogue.example/never-ledger-3.0.1.json", SHA256: strings.Repeat("a", 64),
	})}
	indexData := jsonFixture(t, index)
	var indexRequests, productRequests int
	m.client.Transport = catalogueTransport(func(request *http.Request) (*http.Response, error) {
		var data []byte
		switch request.URL.String() {
		case source:
			indexRequests++
			data = indexData
		case productURL:
			productRequests++
			data = product
		default:
			if strings.HasPrefix(request.URL.String(), registry.server.URL) {
				return http.DefaultTransport.RoundTrip(request)
			}
			t.Fatalf("downloaded unrelated catalogue: %s", request.URL)
		}
		if request.Header.Get("Authorization") != "" {
			t.Fatal("service credentials leaked to a catalogue")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(data)))}, nil
	})
	lock, err := m.Sync(t.Context(), source, target("references"), "ledger", "3.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if lock.Revision != 2 || lock.Catalogue != source || !reflect.DeepEqual(lock.Manifest, latest.Manifest) {
		t.Fatalf("wrong selected release or source: %+v", lock)
	}
	if indexRequests != 1 || productRequests != 1 {
		t.Fatalf("repeated downloads: index=%d product=%d", indexRequests, productRequests)
	}
	m.client.Transport = catalogueTransport(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("offline cache tried a download: %s", request.URL)
		return nil, errors.New("offline")
	})
	cached, err := m.Load(target("references"), "ledger")
	if err != nil || !reflect.DeepEqual(cached, lock) {
		t.Fatalf("cached metadata lost: %+v %v", cached, err)
	}
	if _, err := m.Binary(cached); err != nil {
		t.Fatal(err)
	}
}

func TestReferencedCatalogueRejectsTamperingAndWrongIdentity(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	release := publish(t, m, registry, "3.0.0", 1)
	valid := jsonFixture(t, Catalogue{SchemaVersion: SchemaVersion, Releases: []Release{release}})
	wrongService := release
	wrongService.Service = "auth"
	wrongService.Manifest.Name, wrongService.Manifest.Service, wrongService.Manifest.Root.Use = "auth", "auth", "auth"
	wrongVersion := release
	wrongVersion.ServiceVersion, wrongVersion.Manifest.Version = "3.0.1", "3.0.1"
	for _, test := range []struct {
		name, want string
		data       []byte
		badHash    bool
	}{
		{"tampered", "SHA256 mismatch", []byte("{not even json"), true},
		{"wrong-service", "different service or version", jsonFixture(t, Catalogue{SchemaVersion: SchemaVersion, Releases: []Release{wrongService}}), false},
		{"wrong-version", "different service or version", jsonFixture(t, Catalogue{SchemaVersion: SchemaVersion, Releases: []Release{wrongVersion}}), false},
		{"nested", "nested references", []byte(`{"schemaVersion":2,"plugins":{}}`), false},
		{"unknown-field", "decode referenced catalogue", []byte(`{"schemaVersion":1,"releases":[],"typo":true}`), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			index := referenceIndex("3.0.0", "https://catalogue.example/product.json", test.data)
			if test.badHash {
				index = referenceIndex("3.0.0", "https://catalogue.example/product.json", valid)
			}
			m.client.Transport = catalogueTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(test.data)))}, nil
			})
			_, err := m.ResolveCatalogue(t.Context(), index, "ledger", "3.0.0", 0)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("invalid catalogue accepted or wrong error: %v", err)
			}
			if _, err := m.Load(target(test.name), "ledger"); !errors.Is(err, ErrNotInstalled) {
				t.Fatalf("failed resolution installed a lock: %v", err)
			}
		})
	}
}

func TestRegistryIndexStrictValidationAndMissingVersion(t *testing.T) {
	sha := strings.Repeat("a", 64)
	ref := `{"serviceVersion":"3.0.0","catalogue":"https://example.com/catalogue.json","sha256":"` + sha + `"}`
	for _, test := range []struct{ name, data string }{
		{"missing-plugins", `{"schemaVersion":2}`},
		{"mixed-formats", `{"schemaVersion":2,"plugins":{},"releases":[]}`},
		{"legacy-with-references", `{"schemaVersion":1,"releases":[],"plugins":{}}`},
		{"unknown-field", `{"schemaVersion":2,"plugins":{},"typo":1}`},
		{"duplicate-service", `{"schemaVersion":2,"plugins":{"ledger":{"releases":[` + ref + `]},"ledger":{"releases":[` + ref + `]}}}`},
		{"duplicate-version", `{"schemaVersion":2,"plugins":{"ledger":{"releases":[` + ref + `,` + ref + `]}}}`},
		{"unknown-reference-field", `{"schemaVersion":2,"plugins":{"ledger":{"releases":[` + strings.TrimSuffix(ref, "}") + `,"manifest":{}}]}}}`},
		{"local-path", `{"schemaVersion":2,"plugins":{"ledger":{"releases":[` + strings.Replace(ref, "https://example.com/catalogue.json", "/tmp/catalogue.json", 1) + `]}}}`},
		{"bad-hash", `{"schemaVersion":2,"plugins":{"ledger":{"releases":[` + strings.Replace(ref, sha, "invalid", 1) + `]}}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "registry.json")
			if err := os.WriteFile(path, []byte(test.data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := manager(t).Discover(t.Context(), path); err == nil {
				t.Fatal("invalid registry accepted")
			}
		})
	}
	m := manager(t)
	m.client.Transport = catalogueTransport(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("missing version attempted download: %s", request.URL)
		return nil, errors.New("unexpected download")
	})
	index := referenceIndex("3.0.0", "https://example.com/catalogue.json", []byte("unused"))
	if _, err := m.ResolveCatalogue(t.Context(), index, "ledger", "3.0.1", 0); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("wrong missing version error: %v", err)
	}
	if !index.HasService("ledger", CurrentPlatform()) || index.HasService("auth", CurrentPlatform()) {
		t.Fatal("index advertised the wrong services")
	}
}

func TestReferencedCatalogueExplicitRevisionAndMissingPlatform(t *testing.T) {
	m := manager(t)
	registry := newRegistry(t)
	first := publish(t, m, registry, "3.0.0", 1)
	latest := publish(t, m, registry, "3.0.0", 2)
	data := jsonFixture(t, Catalogue{SchemaVersion: SchemaVersion, Releases: []Release{first, latest}})
	index := referenceIndex("3.0.0", "https://example.com/product.json", data)
	m.client.Transport = catalogueTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(data)))}, nil
	})
	release, err := m.ResolveCatalogue(t.Context(), index, "ledger", "3.0.0", 1)
	if err != nil || release.Revision != 1 {
		t.Fatalf("explicit revision ignored: %+v %v", release, err)
	}
	if _, err := m.ResolveCatalogue(t.Context(), index, "ledger", "3.0.0", 3); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("missing revision resolved: %v", err)
	}
	m.platform = Platform{OS: "linux", Arch: "s390x"}
	if _, err := m.ResolveCatalogue(t.Context(), index, "ledger", "3.0.0", 0); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("missing platform resolved: %v", err)
	}
}
