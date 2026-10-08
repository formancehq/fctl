package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

type providerArgumentsTransport func(*http.Request) (*http.Response, error)

func (transport providerArgumentsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestProviderArgumentsPrefixesWithFlags(t *testing.T) {
	values := []string{"github", "Fixture provider", "fixture-client", "fixture-secret"}
	names := []string{"type", "name", "provider-client-id", "provider-client-secret"}
	for count := range len(values) + 1 {
		t.Run(fmt.Sprintf("prefix-%d", count), func(t *testing.T) {
			flags := map[string]string{"confirm": "true"}
			changed := map[string]bool{}
			for i := count; i < len(names); i++ {
				flags[names[i]], changed[names[i]] = values[i], true
			}
			request := pluginsdk.ExecuteRequest{
				CommandPath: strings.Fields("cloud organizations authentication-provider configure"),
				Args:        slices.Clone(values[:count]), Flags: flags, ChangedFlags: changed,
				Context: map[string]string{"organization": "fixture-org"}, Endpoint: "https://membership.example.test/api",
			}
			checkProviderArgumentsFixture(t, request)
		})
	}
}

// The injected transport handles the complete request in memory. These tests
// never open a socket or write to a real authentication provider.
func checkProviderArgumentsFixture(t *testing.T, request pluginsdk.ExecuteRequest) {
	t.Helper()
	calls := 0
	client := &http.Client{Transport: providerArgumentsTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPut || r.URL.String() != "https://membership.example.test/api/organizations/fixture-org/authentication-provider" {
			t.Errorf("unexpected fixture route: %s %s", r.Method, r.URL)
		}
		body, err := io.ReadAll(r.Body)
		if closeErr := r.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		if err != nil {
			return nil, err
		}
		assertJSON(t, body, []byte(`{"type":"github","name":"Fixture provider","clientID":"fixture-client","clientSecret":"fixture-secret","config":{}}`))
		return &http.Response{
			StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"data":{"name":"Fixture provider"}}`)), Request: r,
		}, nil
	})}
	if _, err := New(client).Execute(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected one fixture request, got %d", calls)
	}
}

func TestProviderArgumentsRejectDuplicateBindings(t *testing.T) {
	values := []string{"github", "Fixture provider", "fixture-client", "fixture-secret"}
	for index, name := range []string{"type", "name", "provider-client-id", "provider-client-secret"} {
		for _, duplicate := range []struct {
			label, value string
			changed      bool
		}{{"same-value", values[index], false}, {"different-value", "conflicting-value", true}, {"explicit-empty", "", true}} {
			t.Run(name+"/"+duplicate.label, func(t *testing.T) {
				request := pluginsdk.ExecuteRequest{
					Args: slices.Clone(values[:index+1]), Flags: map[string]string{name: duplicate.value}, ChangedFlags: map[string]bool{name: duplicate.changed},
				}
				checkProviderArgumentConflict(t, request, name, values[3])
			})
		}
	}
}

func checkProviderArgumentConflict(t *testing.T, request pluginsdk.ExecuteRequest, flag, secret string) {
	t.Helper()
	before := maps.Clone(request.Flags)
	_, err := providerArguments(request)
	if err == nil || !strings.Contains(err.Error(), "argument conflicts with --"+flag) {
		t.Fatalf("duplicate binding must fail for --%s: %v", flag, err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("conflict error exposed the secret argument")
	}
	if !maps.Equal(request.Flags, before) {
		t.Error("conflict handling mutated caller flags")
	}
}

func TestProviderArgumentsRejectTooMany(t *testing.T) {
	_, err := providerArguments(pluginsdk.ExecuteRequest{Args: []string{"github", "name", "client", "fixture-secret", "extra"}})
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("expected an arity error, got %v", err)
	}
}

func TestProviderArgumentsPreserveCallerRequest(t *testing.T) {
	request := pluginsdk.ExecuteRequest{
		Args: []string{"oidc", "Fixture provider"},
		Flags: map[string]string{
			"type": "", "name": "", "provider-client-id": "fixture-client", "provider-client-secret": "fixture-secret", "oidc-issuer": "https://issuer.example.test",
		},
		ChangedFlags: map[string]bool{"provider-client-id": true, "provider-client-secret": true, "oidc-issuer": true},
		Body:         json.RawMessage(`{}`), Context: map[string]string{"organization": "fixture-org"},
	}
	before := request
	before.Flags, before.ChangedFlags, before.Context = maps.Clone(request.Flags), maps.Clone(request.ChangedFlags), maps.Clone(request.Context)
	before.Args, before.Body = slices.Clone(request.Args), slices.Clone(request.Body)
	result, err := providerArguments(request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request, before) {
		t.Fatal("argument binding mutated the caller's request")
	}
	if result.Flags["type"] != "oidc" || result.Flags["name"] != "Fixture provider" || result.Flags["oidc-issuer"] != request.Flags["oidc-issuer"] {
		t.Error("argument binding lost the prefix or the remaining provider flags")
	}
	result.Flags["name"] = "changed"
	if request.Flags["name"] != "" {
		t.Error("bound flags still alias the caller's map")
	}
}

func TestProviderArgumentsPrefixWithJSON(t *testing.T) {
	body, err := providerBody(pluginsdk.ExecuteRequest{
		Args: []string{"oidc", "Fixture provider"},
		Body: json.RawMessage(`{"clientID":"fixture-client","clientSecret":"fixture-secret","config":{"issuer":"https://issuer.example.test"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, body, []byte(`{"type":"oidc","name":"Fixture provider","clientID":"fixture-client","clientSecret":"fixture-secret","config":{"issuer":"https://issuer.example.test"}}`))
}

func TestProviderArgumentsIncompletePrefixStillRequiresFields(t *testing.T) {
	_, err := providerBody(pluginsdk.ExecuteRequest{Args: []string{"github", "Fixture provider"}, Flags: map[string]string{"provider-client-id": "fixture-client"}})
	if err == nil || !strings.Contains(err.Error(), "clientSecret is required") {
		t.Fatalf("missing secret must remain an error: %v", err)
	}
}
