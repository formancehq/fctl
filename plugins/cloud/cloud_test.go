package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

const cloudEnvelope = ` {"data":{"id":"result","amount":123456789012345678901234567890},"cursor":{"next":"opaque"}} `

type cloudRouteCase struct {
	command, method, path, query string
	args                         []string
	flags                        map[string]string
	body, wantBody               string
}

func TestCloudRoutes(t *testing.T) {
	cases := []cloudRouteCase{
		{command: "me info", method: "GET", path: "/api/me"},
		{command: "me invitations list", method: "GET", path: "/api/me/invitations", flags: map[string]string{"status": "pending"}, query: "organization=org&status=pending"},
		{command: "me invitations accept", method: "POST", path: "/api/me/invitations/inv/accept", args: []string{"inv"}},
		{command: "me invitations decline", method: "POST", path: "/api/me/invitations/inv/reject", args: []string{"inv"}, flags: map[string]string{"confirm": "true"}},
		{command: "organizations list", method: "GET", path: "/api/organizations", query: "expand=false"},
		{command: "organizations describe", method: "GET", path: "/api/organizations/other", args: []string{"other"}, query: "expand=false"},
		{command: "organizations create", method: "POST", path: "/api/organizations", args: []string{"Books"}, flags: map[string]string{"domain": "books.test", "default-policy-id": "42", "owner-id": "owner"}, wantBody: `{"name":"Books","domain":"books.test","defaultPolicyID":42,"ownerID":"owner"}`},
		{command: "organizations create", method: "POST", path: "/api/organizations", body: `{"name":"Books","defaultPolicyID":1234567890123456789}`, wantBody: `{"name":"Books","defaultPolicyID":1234567890123456789}`},
		{command: "organizations delete", method: "DELETE", path: "/api/organizations/org", flags: map[string]string{"confirm": "true"}},
		{command: "organizations history", method: "GET", path: "/api/organizations/org/logs", flags: map[string]string{"cursor": "a+b", "page-size": "20"}, query: "cursor=a%2Bb&pageSize=20"},
		{command: "organizations users list", method: "GET", path: "/api/organizations/org/users"},
		{command: "organizations users show", method: "GET", path: "/api/organizations/org/users/user", args: []string{"user"}},
		{command: "organizations users link", method: "PUT", path: "/api/organizations/org/users/user", args: []string{"user"}, flags: map[string]string{"policy-id": "9", "confirm": "true"}, wantBody: `{"policyId":9}`},
		{command: "organizations users unlink", method: "DELETE", path: "/api/organizations/org/users/user", args: []string{"user"}, flags: map[string]string{"confirm": "true"}},
		{command: "organizations invitations list", method: "GET", path: "/api/organizations/org/invitations", flags: map[string]string{"status": "PENDING"}, query: "status=PENDING"},
		{command: "organizations invitations send", method: "POST", path: "/api/organizations/org/invitations", args: []string{"a+b@example.test"}, query: "email=a%2Bb%40example.test"},
		{command: "organizations invitations delete", method: "DELETE", path: "/api/organizations/org/invitations/inv", args: []string{"inv"}, flags: map[string]string{"confirm": "true"}},
		{command: "organizations policies list", method: "GET", path: "/api/organizations/org/policies"},
		{command: "organizations policies show", method: "GET", path: "/api/organizations/org/policies/2", args: []string{"2"}},
		{command: "organizations policies create", method: "POST", path: "/api/organizations/org/policies", flags: map[string]string{"name": "readers", "description": "Read only"}, wantBody: `{"name":"readers","description":"Read only"}`},
		{command: "organizations policies delete", method: "DELETE", path: "/api/organizations/org/policies/2", args: []string{"2"}, flags: map[string]string{"confirm": "true"}},
		{command: "organizations policies add-scope", method: "PUT", path: "/api/organizations/org/policies/2/scopes/3", args: []string{"2", "3"}, flags: map[string]string{"confirm": "true"}},
		{command: "organizations policies remove-scope", method: "DELETE", path: "/api/organizations/org/policies/2/scopes/3", args: []string{"2", "3"}, flags: map[string]string{"confirm": "true"}},
		{command: "organizations oauth-clients list", method: "GET", path: "/api/organizations/org/clients"},
		{command: "organizations oauth-clients show", method: "GET", path: "/api/organizations/org/clients/client", args: []string{"client"}},
		{command: "organizations oauth-clients create", method: "POST", path: "/api/organizations/org/clients", flags: map[string]string{"name": "reader"}, wantBody: `{"name":"reader"}`},
		{command: "organizations oauth-clients delete", method: "DELETE", path: "/api/organizations/org/clients/client", args: []string{"client"}, flags: map[string]string{"confirm": "true"}},
		{command: "organizations authentication-provider show", method: "GET", path: "/api/organizations/org/authentication-provider"},
		{command: "organizations authentication-provider delete", method: "DELETE", path: "/api/organizations/org/authentication-provider", flags: map[string]string{"confirm": "true"}},
		{command: "organizations authentication-provider configure", method: "PUT", path: "/api/organizations/org/authentication-provider", args: []string{"github", "GitHub", "client", "secret"}, flags: map[string]string{"confirm": "true"}, wantBody: `{"type":"github","name":"GitHub","clientID":"client","clientSecret":"secret","config":{}}`},
		{command: "organizations authentication-provider configure", method: "PUT", path: "/api/organizations/org/authentication-provider", flags: map[string]string{"confirm": "true", "type": "oidc", "name": "SSO", "provider-client-id": "client", "provider-client-secret": "secret", "oidc-issuer": "https://idp.test"}, wantBody: `{"type":"oidc","name":"SSO","clientID":"client","clientSecret":"secret","config":{"issuer":"https://idp.test"}}`},
		{command: "organizations applications list", method: "GET", path: "/api/organizations/org/applications", query: "expand=false"},
		{command: "organizations applications show", method: "GET", path: "/api/organizations/org/applications/app", args: []string{"app"}},
		{command: "organizations applications enable", method: "PUT", path: "/api/organizations/org/applications/app", args: []string{"app"}},
		{command: "organizations applications disable", method: "DELETE", path: "/api/organizations/org/applications/app", args: []string{"app"}, flags: map[string]string{"confirm": "true"}},
		{command: "regions list", method: "GET", path: "/api/organizations/org/regions"},
		{command: "regions show", method: "GET", path: "/api/organizations/org/regions/eu", args: []string{"eu"}},
		{command: "regions versions", method: "GET", path: "/api/organizations/org/regions/eu/versions", args: []string{"eu"}},
		{command: "regions create", method: "POST", path: "/api/organizations/org/regions", flags: map[string]string{"name": "private"}, wantBody: `{"name":"private"}`},
		{command: "regions create", method: "POST", path: "/api/organizations/org/regions", args: []string{"private"}, wantBody: `{"name":"private"}`},
		{command: "regions delete", method: "DELETE", path: "/api/organizations/org/regions/eu", args: []string{"eu"}, flags: map[string]string{"confirm": "true"}},
		{command: "stack list", method: "GET", path: "/api/organizations/org/stacks"},
	}
	for _, tc := range cases {
		t.Run(tc.command+" "+strings.Join(tc.args, " ")+tc.body, func(t *testing.T) {
			checkCloudRoute(t, tc)
		})
	}
}
func assertJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var a, b any
	d := json.NewDecoder(strings.NewReader(string(got)))
	d.UseNumber()
	if err := d.Decode(&a); err != nil {
		t.Fatalf("invalid JSON %s: %v", got, err)
	}
	d = json.NewDecoder(strings.NewReader(string(want)))
	d.UseNumber()
	if err := d.Decode(&b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("JSON=%s, want %s", got, want)
	}
}

func TestCloudUpdatePreservesOmittedFields(t *testing.T) {
	for _, tc := range []cloudUpdateCase{
		{"organizations update", "org", `{"data":{"name":"old","domain":"books.test","defaultPolicyID":1234567890123456789,"id":"org"}}`, `{"name":"new","domain":"books.test","defaultPolicyID":1234567890123456789}`},
		{"organizations policies update", "2", `{"data":{"name":"old","description":"retained","id":2,"protected":false}}`, `{"name":"new","description":"retained"}`},
		{"organizations oauth-clients update", "client", `{"data":{"name":"old","description":"retained","secret":{"clear":"hidden"}}}`, `{"name":"new","description":"retained"}`},
	} {
		t.Run(tc.command, func(t *testing.T) {
			checkCloudUpdate(t, tc)
		})
	}
}
func TestCloudOrgSelectionAndIsolation(t *testing.T) {
	for _, tc := range []cloudSelectionCase{
		{"selected", nil, map[string]string{"organization": "chosen"}, "chosen"},
		{"default", nil, map[string]string{"organization": "default"}, "default"},
		{"unique", nil, map[string]string{"organizations": `["only"]`}, "only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkCloudSelection(t, tc)
		})
	}
	for _, ids := range []string{`[]`, `["b","a"]`, `not json`} {
		req := pluginsdk.ExecuteRequest{Context: map[string]string{"organizations": ids}}
		_, err := orgID(req)
		if err == nil {
			t.Fatalf("accepted ambiguous/invalid organizations %s", ids)
		}
		if ids == `["b","a"]` && (!strings.Contains(err.Error(), "cloud organizations list") || strings.Contains(err.Error(), "a, b")) {
			t.Errorf("organization selection error must give concise discovery instructions: %v", err)
		}
	}
}
func TestCloudOrganizationDiscovery(t *testing.T) {
	for _, tc := range []cloudDiscoveryCase{{`{"data":[{"id":"only"}]}`, true}, {`{"data":[{"id":"a"},{"id":"b"}]}`, false}, {`{"data":[]}`, false}} {
		t.Run(tc.response, func(t *testing.T) {
			checkCloudDiscovery(t, tc)
		})
	}
}
func TestCloudRejectsInvalidRequestsBeforeHTTP(t *testing.T) {
	cases := []pluginsdk.ExecuteRequest{
		{CommandPath: []string{"cloud", "regions", "list"}, Flags: map[string]string{"organization": "org"}},
		{CommandPath: []string{"cloud", "stack", "list"}, Flags: map[string]string{"stack": "stack"}},
		{CommandPath: []string{"cloud", "organizations", "oauth-clients", "create"}, Flags: map[string]string{"data": "@unread.json"}},
		{CommandPath: []string{"cloud", "regions", "create"}, Body: json.RawMessage(`{"name":12}`)},
		{CommandPath: []string{"cloud", "regions", "create"}, Body: json.RawMessage(`{"name":"   "}`)},
		{CommandPath: []string{"cloud", "organizations", "users", "link"}, Args: []string{"user"}, Flags: map[string]string{"confirm": "true"}, Body: json.RawMessage(`{"policyId":"9"}`)},
		{CommandPath: []string{"cloud", "organizations", "authentication-provider", "configure"}, Flags: map[string]string{"confirm": "true"}, Body: json.RawMessage(`{"type":"other","name":"SSO","clientID":"client","clientSecret":"secret"}`)},
		{CommandPath: []string{"cloud", "organizations", "authentication-provider", "configure"}, Flags: map[string]string{"confirm": "true"}, Body: json.RawMessage(`{"type":"oidc","name":"SSO","clientID":"client","clientSecret":"secret","config":{}}`)},

		{CommandPath: []string{"cloud", "regions", "delete"}, Args: []string{"eu"}},
		{CommandPath: []string{"cloud", "organizations", "delete"}},
		{CommandPath: []string{"cloud", "me", "invitations", "decline"}, Args: []string{"inv"}},
		{CommandPath: []string{"cloud", "regions", "create"}, Flags: map[string]string{"data": "@unread.json"}},
		{CommandPath: []string{"cloud", "regions", "create"}, Body: json.RawMessage(`[]`)},
		{CommandPath: []string{"cloud", "regions", "create"}, Body: json.RawMessage(`null`)},
		{CommandPath: []string{"cloud", "regions", "create"}, Flags: map[string]string{"name": "flag"}, Body: json.RawMessage(`{"name":"body"}`)},
		{CommandPath: []string{"cloud", "organizations", "history"}, Flags: map[string]string{"page-size": "0"}},
		{CommandPath: []string{"cloud", "organizations", "policies", "show"}, Args: []string{"not-int"}},
		{CommandPath: []string{"cloud", "organizations", "users", "link"}, Args: []string{"user"}, Flags: map[string]string{"confirm": "true", "policy-id": "0"}},
		{CommandPath: []string{"cloud", "organizations", "invitations", "send"}},
		{CommandPath: []string{"cloud", "regions", "show"}, Args: []string{".."}},
		{CommandPath: []string{"cloud", "regions", "list"}, Flags: map[string]string{"unknown": "bad"}},
	}
	for _, req := range cases {
		t.Run(strings.Join(req.CommandPath, " ")+string(req.Body), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("invalid request reached server") }))
			defer server.Close()
			req.Context = map[string]string{"organization": "org"}
			req.Endpoint = server.URL + "/api"
			if _, err := New(server.Client()).Execute(t.Context(), req); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}
func TestCloudManifestProtocol(t *testing.T) {
	p := New(nil)
	m, err := p.GetManifest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "cloud" || m.Service != "cloud" || m.ProtocolVersion != pluginsdk.ProtocolVersion || m.Version == "" {
		t.Fatalf("identity: %#v", m)
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pluginsdk.Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, decoded) {
		t.Fatal("manifest failed JSON round trip")
	}
	checkCloudManifestCommand(t, m, m.Root, nil)
	m.Root.Short = "mutated"
	fresh, err := p.GetManifest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Root.Short == "mutated" {
		t.Fatal("manifest leaked mutable state")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := p.GetManifest(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("manifest cancellation: %v", err)
	}
}
func TestCloudEscapesIdentifiers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/organizations/org/regions/a%2Fb%3F%23" {
			t.Errorf("escaped path=%s", r.URL.EscapedPath())
		}
		cloudTestWrite(t, w, cloudEnvelope)
	}))
	defer server.Close()
	_, err := New(server.Client()).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"cloud", "regions", "show"}, Args: []string{"a/b?#"}, Context: map[string]string{"organization": "org"}, Endpoint: server.URL + "/api"})
	if err != nil {
		t.Fatal(err)
	}
}
func TestCloudImportClosure(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "list", "-deps", "-json", "github.com/formancehq/fctl/v4/plugins/cloud")
	data, err := command.Output()
	if err != nil {
		t.Fatalf("inspect import closure: %v", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var pkg struct {
			ImportPath string
			Standard   bool
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !pkg.Standard && pkg.ImportPath != "github.com/formancehq/fctl/v4/plugins/cloud" && !strings.HasPrefix(pkg.ImportPath, "github.com/formancehq/fctl/pkg/pluginsdk") {
			t.Errorf("plugin has non-public or non-stdlib dependency: %s", pkg.ImportPath)
		}
	}
}

func TestCloudHTTPErrorAndEmptyResponse(t *testing.T) {
	for _, status := range []int{204, 403} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			checkCloudHTTPStatus(t, status)
		})
	}
}

func TestCloudUpdatesExplicitEmptyFlags(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			cloudTestWrite(t, w, `{"data":{"name":"Books","domain":"old.test","defaultPolicyID":4}}`)
			return
		}
		body := cloudTestRead(t, r.Body)
		assertJSON(t, body, []byte(`{"name":"Books","domain":"","defaultPolicyID":4}`))
		cloudTestWrite(t, w, cloudEnvelope)
	}))
	defer server.Close()
	req := pluginsdk.ExecuteRequest{CommandPath: []string{"cloud", "organizations", "update"}, Flags: map[string]string{"confirm": "true", "domain": ""}, ChangedFlags: map[string]bool{"domain": true}, Context: map[string]string{"organization": "org"}, Endpoint: server.URL + "/api"}
	if _, err := New(server.Client()).Execute(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}

type organizationPolicyUpdateCase struct {
	name, current, patch, flag, wantBody string
	wantGets                             int32
	wantError                            bool
}

func TestCloudOrganizationUpdateDefaultPolicy(t *testing.T) {
	for _, tc := range []organizationPolicyUpdateCase{
		{name: "unset omitted", current: `{}`, patch: `{"name":"new","domain":"new.test"}`, wantGets: 1, wantError: true},
		{name: "null omitted", current: `{"defaultPolicyID":null}`, patch: `{"name":"new","domain":"new.test"}`, wantGets: 1, wantError: true},
		{name: "zero omitted", current: `{"defaultPolicyID":0}`, patch: `{"name":"new","domain":"new.test"}`, wantGets: 1, wantError: true},
		{name: "explicit null", current: `{"defaultPolicyID":4}`, patch: `{"name":"new","domain":"new.test","defaultPolicyID":null}`, wantError: true},
		{name: "explicit zero", current: `{"defaultPolicyID":4}`, patch: `{"name":"new","domain":"new.test","defaultPolicyID":0}`, wantError: true},
		{name: "zero flag", current: `{"defaultPolicyID":4}`, flag: "0", patch: `{"name":"new"}`, wantError: true},
		{name: "positive preserved", current: `{"defaultPolicyID":1234567890123456789}`, patch: `{"name":"new"}`, wantGets: 1, wantBody: `{"name":"new","domain":"old.test","defaultPolicyID":1234567890123456789}`},
		{name: "positive data from null", current: `{"defaultPolicyID":null}`, patch: `{"defaultPolicyID":9}`, wantGets: 1, wantBody: `{"name":"old","domain":"old.test","defaultPolicyID":9}`},
		{name: "positive flag changes", current: `{"defaultPolicyID":4}`, flag: "9", patch: `{"name":"new"}`, wantGets: 1, wantBody: `{"name":"new","domain":"old.test","defaultPolicyID":9}`},
	} {
		t.Run(tc.name, func(t *testing.T) { checkOrganizationPolicyUpdate(t, tc) })
	}
}

func checkOrganizationPolicyUpdate(t *testing.T, tc organizationPolicyUpdateCase) {
	t.Helper()
	var gets, puts atomic.Int32
	current := map[string]json.RawMessage{"name": json.RawMessage(`"old"`), "domain": json.RawMessage(`"old.test"`)}
	var policy map[string]json.RawMessage
	if err := json.Unmarshal([]byte(tc.current), &policy); err != nil {
		t.Fatal(err)
	}
	maps.Copy(current, policy)
	original := maps.Clone(current)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/organizations/org" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			gets.Add(1)
			if err := json.NewEncoder(w).Encode(map[string]any{"data": current}); err != nil {
				t.Error(err)
			}
		case http.MethodPut:
			puts.Add(1)
			assertJSON(t, cloudTestRead(t, r.Body), []byte(tc.wantBody))
			cloudTestWrite(t, w, cloudEnvelope)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	flags := map[string]string{"confirm": "true"}
	if tc.flag != "" {
		flags["default-policy-id"] = tc.flag
	}
	req := pluginsdk.ExecuteRequest{CommandPath: []string{"cloud", "organizations", "update"}, Flags: flags, Body: json.RawMessage(tc.patch), Context: map[string]string{"organization": "org"}, Endpoint: server.URL + "/api"}
	_, err := New(server.Client()).Execute(t.Context(), req)
	checkOrganizationPolicyUpdateResult(t, tc, err, gets.Load(), puts.Load())
	if !reflect.DeepEqual(current, original) {
		t.Fatal("current organization fields changed")
	}
}

func checkOrganizationPolicyUpdateResult(t *testing.T, tc organizationPolicyUpdateCase, err error, gets, puts int32) {
	t.Helper()
	if (err != nil) != tc.wantError {
		t.Fatalf("error=%v, wantError=%v", err, tc.wantError)
	}
	if tc.wantError && (!strings.Contains(err.Error(), "--default-policy-id") || !strings.Contains(err.Error(), "--data")) {
		t.Errorf("error lacks explicit policy guidance: %v", err)
	}
	wantPuts := int32(1)
	if tc.wantError {
		wantPuts = 0
	}
	if gets != tc.wantGets || puts != wantPuts {
		t.Errorf("GET=%d PUT=%d, want GET=%d PUT=%d", gets, puts, tc.wantGets, wantPuts)
	}
}
func TestCloudExecuteBoundaryFailures(t *testing.T) {
	req := pluginsdk.ExecuteRequest{CommandPath: []string{"cloud", "me", "info"}, Endpoint: "https://membership.test"}
	if _, err := New(nil).Execute(t.Context(), req); err == nil {
		t.Fatal("nil client accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := New(nil).Execute(ctx, req); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation=%v", err)
	}
	req.Endpoint = "https://user:password@membership.test"
	if _, err := New(&http.Client{}).Execute(t.Context(), req); err == nil {
		t.Fatal("endpoint credentials accepted")
	}
}
func TestCloudAllConfirmLeavesRejectBeforeClient(t *testing.T) {
	m := manifest()
	var walk func(pluginsdk.CommandSpec, []string)
	walk = func(spec pluginsdk.CommandSpec, path []string) {
		path = append(path, pluginsdk.CommandName(spec))
		if spec.Confirm {
			req := pluginsdk.ExecuteRequest{CommandPath: path, Flags: cloudContractFlags(t, m, path), Body: json.RawMessage(`{}`)}
			for range spec.Args.Min {
				req.Args = append(req.Args, "id")
			}
			_, err := New(nil).Execute(t.Context(), req)
			if err == nil || !strings.Contains(err.Error(), "requires --confirm") {
				t.Errorf("%v bypassed confirmation: %v", path, err)
			}
		}
		for _, sub := range spec.Subcommands {
			walk(sub, append([]string(nil), path...))
		}
	}
	walk(m.Root, nil)
}

// Supply required leaf fields and inherited opt-in guards when exercising a manifest.
func cloudContractFlags(t *testing.T, m pluginsdk.Manifest, path []string) map[string]string {
	t.Helper()
	flags := make(map[string]string)
	for i := range path {
		spec, err := pluginsdk.FindCommand(m, path[:i+1])
		if err != nil {
			t.Fatal(err)
		}
		for _, flag := range spec.Flags {
			if i != len(path)-1 && !flag.Persistent {
				continue
			}
			if flag.RequireTrue {
				flags[flag.Name] = "true"
			} else if flag.Required && !flag.Body {
				flags[flag.Name] = "id"
			}
		}
	}
	return flags
}
func TestCloudAppsIntegratedDispatch(t *testing.T) {
	m := manifest()
	path := []string{"cloud", "apps", "list"}
	service, err := pluginsdk.CommandService(m, path)
	if err != nil || service != "cloud-apps" {
		t.Fatalf("apps service=%s err=%v", service, err)
	}
	// Missing opt-in must fail before requiring an authenticated client.
	if _, err := New(nil).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: path}); err == nil || !strings.Contains(err.Error(), "experimental") {
		t.Fatalf("experimental guard=%v", err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/deploy/apps" || r.URL.RawQuery != "pageSize=100" {
			t.Errorf("Deploy request=%s", r.URL)
		}
		cloudTestWrite(t, w, cloudEnvelope)
	}))
	defer server.Close()
	response, err := New(server.Client()).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: path, Flags: map[string]string{"experimental": "true"}, Endpoint: server.URL + "/deploy"})
	if err != nil || calls != 1 || string(response.Data) != cloudEnvelope {
		t.Fatalf("apps data=%s err=%v calls=%d", response.Data, err, calls)
	}
}

func checkCloudRoute(t *testing.T, tc cloudRouteCase) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkCloudRouteRequest(t, tc, &calls, w, r)
	}))
	defer server.Close()
	req := pluginsdk.ExecuteRequest{CommandPath: append([]string{"cloud"}, strings.Fields(tc.command)...), Args: tc.args, Flags: tc.flags, Context: map[string]string{"organization": "org"}, Endpoint: server.URL + "/issuer/api"}
	if tc.body != "" {
		req.Body = json.RawMessage(tc.body)
	}
	response, err := New(server.Client()).Execute(t.Context(), req)
	if err != nil || string(response.Data) != cloudEnvelope || calls != 1 {
		t.Fatalf("response=%s err=%v calls=%d", response.Data, err, calls)
	}

}

type cloudUpdateCase struct{ command, id, current, want string }

func checkCloudUpdate(t *testing.T, tc cloudUpdateCase) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			if r.Method != "GET" {
				t.Error("update must read resource first")
			}
			cloudTestWrite(t, w, tc.current)
			return
		}
		if r.Method != "PUT" {
			t.Error("update must PUT")
		}
		body := cloudTestRead(t, r.Body)
		assertJSON(t, body, []byte(tc.want))
		cloudTestWrite(t, w, cloudEnvelope)
	}))
	defer server.Close()
	req := pluginsdk.ExecuteRequest{CommandPath: append([]string{"cloud"}, strings.Fields(tc.command)...), Args: []string{tc.id}, Flags: map[string]string{"confirm": "true", "name": "new"}, Context: map[string]string{"organization": "org"}, Endpoint: server.URL + "/api"}
	_, err := New(server.Client()).Execute(t.Context(), req)
	if err != nil || calls != 2 {
		t.Fatalf("update calls=%d error=%v", calls, err)
	}

}

type cloudSelectionCase struct {
	name       string
	flags, ctx map[string]string
	want       string
}

func checkCloudSelection(t *testing.T, tc cloudSelectionCase) {
	flags, host := maps.Clone(tc.flags), maps.Clone(tc.ctx)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/organizations/"+tc.want+"/regions" {
			t.Errorf("wrong target: %s", r.URL)
		}
		cloudTestWrite(t, w, cloudEnvelope)
	}))
	defer server.Close()
	req := pluginsdk.ExecuteRequest{CommandPath: []string{"cloud", "regions", "list"}, Flags: tc.flags, Context: tc.ctx, Endpoint: server.URL + "/api"}
	if _, err := New(server.Client()).Execute(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(tc.flags, flags) || !maps.Equal(tc.ctx, host) {
		t.Fatal("plugin mutated caller context or flags")
	}

}

type cloudDiscoveryCase struct {
	response string
	success  bool
}

func checkCloudDiscovery(t *testing.T, tc cloudDiscoveryCase) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			if r.URL.Path != "/api/organizations" {
				t.Errorf("discovery route: %s", r.URL)
			}
			cloudTestWrite(t, w, tc.response)
			return
		}
		if r.URL.Path != "/api/organizations/only/regions" {
			t.Error("wrong discovered organization")
		}
		cloudTestWrite(t, w, cloudEnvelope)
	}))
	defer server.Close()
	_, err := New(server.Client()).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"cloud", "regions", "list"}, Endpoint: server.URL + "/api"})
	if tc.success {
		if err != nil || calls != 2 {
			t.Fatalf("discovery err=%v calls=%d", err, calls)
		}
	} else if err == nil || calls != 1 {
		t.Fatalf("ambiguous discovery err=%v calls=%d", err, calls)
	}

}
func checkCloudHTTPStatus(t *testing.T, status int) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if status == 403 {
			cloudTestWrite(t, w, `{"errorCode":"FORBIDDEN","errorMessage":"denied"}`)
		}
	}))
	defer server.Close()
	response, err := New(server.Client()).Execute(t.Context(), pluginsdk.ExecuteRequest{CommandPath: []string{"cloud", "me", "info"}, Endpoint: server.URL + "/api"})
	if status == 204 {
		if err != nil || string(response.Data) != "null" {
			t.Fatalf("empty=%s err=%v", response.Data, err)
		}
	} else {
		if err == nil || len(response.Data) != 0 || !strings.Contains(err.Error(), "FORBIDDEN") {
			t.Fatalf("failure=%s err=%v", response.Data, err)
		}
	}

}

func checkCloudRouteRequest(t *testing.T, tc cloudRouteCase, calls *int, w http.ResponseWriter, r *http.Request) {
	(*calls)++
	if r.Method != tc.method || r.URL.Path != "/issuer"+tc.path || r.URL.RawQuery != tc.query {
		t.Errorf("request = %s %s, want %s /issuer%s?%s", r.Method, r.URL, tc.method, tc.path, tc.query)
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if tc.wantBody == "" {
		if len(data) != 0 {
			t.Errorf("unexpected body: %s", data)
		}
	} else {
		assertJSON(t, data, []byte(tc.wantBody))
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing JSON content type")
		}
	}
	if r.Header.Get("Accept") != "application/json" {
		t.Error("missing JSON accept")
	}
	cloudTestWrite(t, w, cloudEnvelope)

}

func checkCloudManifestCommand(t *testing.T, m pluginsdk.Manifest, c pluginsdk.CommandSpec, path []string) {
	t.Helper()
	path = append(path, pluginsdk.CommandName(c))
	if c.Runnable {
		req := pluginsdk.ExecuteRequest{CommandPath: path, Flags: cloudContractFlags(t, m, path), Body: json.RawMessage(`{}`)}
		for range c.Args.Min {
			req.Args = append(req.Args, "id")
		}
		if c.Confirm {
			req.Flags["confirm"] = "true"
		}
		if _, err := pluginsdk.NormalizeRequest(m, req); err != nil {
			t.Errorf("invalid manifest leaf %v: %v", path, err)
		}
	}
	for _, sub := range c.Subcommands {
		checkCloudManifestCommand(t, m, sub, append([]string(nil), path...))
	}
}
func cloudTestWrite(t *testing.T, w io.Writer, value string) {
	t.Helper()
	if _, err := io.WriteString(w, value); err != nil {
		t.Error(err)
	}
}
func cloudTestRead(t *testing.T, r io.Reader) []byte {
	t.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	return data
}
