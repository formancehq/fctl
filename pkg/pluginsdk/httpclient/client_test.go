package httpclient_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func TestRequestPreservesPathQueryAndNumbers(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/gateway/v3/books/accounts/users%2F001" {
			t.Errorf("path: %s", r.URL.EscapedPath())
		}
		if r.URL.Query().Get("cursor") != "opaque +/= " {
			t.Errorf("query: %s", r.URL.RawQuery)
		}
		if r.Header.Get("Idempotency-Key") != "request-1" {
			t.Error("missing idempotency key")
		}
		_, err := w.Write([]byte(`{"amount":90071992547409930001}`))
		if err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := httpclient.New(server.URL+"/gateway", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(t.Context(), http.MethodPost, httpclient.Path("v3", "books", "accounts", "users/001"), url.Values{"cursor": {"opaque +/= "}}, json.RawMessage(`{"amount":90071992547409930001}`), http.Header{"Idempotency-Key": {"request-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(response) != `{"amount":90071992547409930001}` {
		t.Fatalf("numbers changed: %s", response)
	}
}

//nolint:gocognit // Both explicit merge-patch and default JSON paths must preserve caller headers.
func TestExplicitJSONMediaType(t *testing.T) {
	t.Parallel()
	for _, contentType := range []string{"application/merge-patch+json", "application/json"} {
		t.Run(contentType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != contentType || r.Header.Get("Accept") != "application/json" {
					t.Errorf("headers=%v", r.Header)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)
			client, err := httpclient.New(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			headers := http.Header{}
			if contentType != "application/json" {
				headers.Set("Content-Type", contentType)
			}
			if _, err := client.Do(t.Context(), http.MethodPatch, "/resource", nil, json.RawMessage(`{"value":null}`), headers); err != nil {
				t.Fatal(err)
			}
			if contentType == "application/json" && headers.Get("Content-Type") != "" {
				t.Fatal("request mutated caller headers")
			}
		})
	}
}

func TestErrorsDoNotRetry(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(status)
				_, err := w.Write([]byte(`{"errorCode":"TEST_FAILURE","errorMessage":"rejected"}`))
				if err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			client, err := httpclient.New(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Do(t.Context(), http.MethodPost, "/v3/test", nil, nil, nil)
			if err == nil || !strings.Contains(err.Error(), "TEST_FAILURE") {
				t.Fatalf("error: %v", err)
			}
			if calls != 1 {
				t.Fatalf("mutation executed %d times", calls)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"localhost:9000", "file:///tmp/data", "https://user:secret@service", "https://service?token=secret", "https://service#fragment", ""} {
		if httpclient.ValidateURL(value) == nil {
			t.Errorf("accepted %q", value)
		}
	}
	if err := httpclient.ValidateURL("http://localhost:9000/prefix"); err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticatedEndpointsRequireTLS(t *testing.T) {
	t.Parallel()
	for _, endpoint := range []string{"http://ledger.example.com", "http://192.0.2.1/token"} {
		if httpclient.ValidateSecureURL(endpoint) == nil {
			t.Errorf("accepted insecure endpoint %s", endpoint)
		}
	}
	for _, endpoint := range []string{"https://ledger.example.com", "http://localhost:9000", "http://127.0.0.1:9000", "http://[::1]:9000"} {
		if err := httpclient.ValidateSecureURL(endpoint); err != nil {
			t.Errorf("rejected %s: %v", endpoint, err)
		}
	}
}

func TestEmptyAndInvalidResponses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		body string
		want string
		fail bool
	}{{"", "null", false}, {"<html>proxy failure</html>", "", true}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(tc.body))
			if err != nil {
				t.Error(err)
			}
		}))
		client, err := httpclient.New(server.URL, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		result, err := client.Do(t.Context(), http.MethodGet, "/", nil, nil, nil)
		server.Close()
		if (err != nil) != tc.fail || string(result) != tc.want {
			t.Errorf("result=%s error=%v", result, err)
		}
	}
}

func TestWithContextClonesClientAndMetadata(t *testing.T) {
	t.Parallel()
	transport := &http.Client{}
	base, err := httpclient.New("https://ledger.example/prefix", transport)
	if err != nil {
		t.Fatal(err)
	}
	if base.Context() != nil {
		t.Fatal("new client has metadata by default")
	}
	values := map[string]string{"organization": "org-1", "stack": "stack-1", "organizationIDs": `["org-1","org-2"]`}
	want := maps.Clone(values)
	client := base.WithContext(values)
	if client == base || client.Endpoint() != base.Endpoint() || client.HTTPClient() != transport || base.Context() != nil {
		t.Fatal("WithContext changed the original client or its HTTP transport")
	}
	values["organization"] = "host-change"
	if !maps.Equal(client.Context(), want) {
		t.Fatal("client metadata shares the caller map")
	}
	view := client.Context()
	view["stack"] = "plugin-change"
	delete(view, "organizationIDs")
	if !maps.Equal(client.Context(), want) {
		t.Fatal("Context exposed the client's metadata map")
	}
	replacement := client.WithContext(map[string]string{"organization": "org-2"})
	if replacement.Context()["organization"] != "org-2" || len(replacement.Context()) != 1 || !maps.Equal(client.Context(), want) {
		t.Fatal("derived context merged metadata or changed its parent")
	}
	if cleared := client.WithContext(nil); cleared.Context() != nil || !maps.Equal(client.Context(), want) {
		t.Fatal("clearing derived metadata changed its parent")
	}
}

func TestContextDoesNotEnterHTTPRequests(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/v3/" || r.Header.Get("Organization") != "" || r.Header.Get("Stack") != "" {
			t.Error("host metadata was added to the HTTP request")
		}
		if _, err := w.Write([]byte(`{"data":[]}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := httpclient.New(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client = client.WithContext(map[string]string{"organization": "org-1", "stack": "stack-1"})
	if _, err := client.Do(t.Context(), http.MethodGet, "/v3/", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}
