package plugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk/httpclient"
)

func multiServiceManifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{Name: "multi", Version: "test", Service: "default", ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{
		Use: "multi", Short: "Multi service fixture", Runnable: true, Service: "root",
		Subcommands: []pluginsdk.CommandSpec{
			{Use: "inherited", Runnable: true},
			{Use: "branch", Service: "branch", Subcommands: []pluginsdk.CommandSpec{
				{Use: "inherited", Runnable: true}, {Use: "leaf", Service: "leaf", Runnable: true},
			}},
		},
	}}
}

func attachRequest(t *testing.T, manifest pluginsdk.Manifest, resolver plugin.RequestResolver, factory plugin.Factory) *cobra.Command {
	t.Helper()
	registry := &plugin.Registry{}
	if err := registry.Register(t.Context(), &fakePlugin{manifest: manifest}, factory); err != nil {
		t.Fatal(err)
	}
	root := &cobra.Command{Use: "fctl", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().String("connection", "", "host setting")
	if err := plugin.NewCommandWithRequest(registry, resolver).AddTo(root); err != nil {
		t.Fatal(err)
	}
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root
}

func TestRequestResolverRejectsBeforeResolution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"experimental-omitted", []string{"ledger", "write", "fixture", "--tenant=t", "--confirm"}},
		{"experimental-false", []string{"ledger", "write", "fixture", "--tenant=t", "--confirm", "--experimental=false"}},
		{"confirm-false", []string{"ledger", "write", "fixture", "--tenant=t", "--experimental", "--confirm=false"}},
		{"confirm-omitted", []string{"ledger", "write", "fixture", "--tenant=t", "--experimental"}},
		{"bad-body", []string{"ledger", "write", "fixture", "--tenant=t", "--experimental", "--confirm", "--data={"}},
		{"bad-type", []string{"ledger", "write", "fixture", "--tenant=t", "--experimental", "--confirm", "--limit=4294967296"}},
		{"missing-argument", []string{"ledger", "write", "--tenant=t", "--experimental", "--confirm"}},
		{"unknown-path", []string{"unknown"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := testManifest()
			manifest.Root.Flags = append(manifest.Root.Flags, pluginsdk.FlagSpec{Name: "experimental", Type: "bool", Default: "false", RequireTrue: true, Persistent: true})
			calls, factories := 0, 0
			root := attachRequest(t, manifest, func(context.Context, string, pluginsdk.ExecuteRequest) (*api.Client, error) { calls++; return nil, nil }, func(*http.Client) pluginsdk.Plugin { factories++; return nil })
			root.SetArgs(tc.args)
			if err := root.ExecuteContext(t.Context()); err == nil {
				t.Fatal("invalid request accepted")
			}
			if calls != 0 || factories != 0 {
				t.Fatalf("rejected request resolved %d times and constructed %d plugins", calls, factories)
			}
		})
	}
}

func TestRegistryRejectsNonBooleanRequireTrue(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"string", "uint32"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			manifest := multiServiceManifest()
			manifest.Root.Flags = []pluginsdk.FlagSpec{{Name: "experimental", Type: kind, Default: "0", RequireTrue: true}}
			registry := &plugin.Registry{}
			err := registry.Register(t.Context(), &fakePlugin{manifest: manifest}, func(*http.Client) pluginsdk.Plugin { t.Fatal("registration constructed a plugin"); return nil })
			if err == nil || !strings.Contains(err.Error(), "boolean") {
				t.Fatalf("invalid gate declaration: %v", err)
			}
		})
	}
}

func TestOnePluginRoutesMultipleServices(t *testing.T) {
	t.Parallel()
	manifest := multiServiceManifest()
	clients := map[string]*api.Client{}
	for _, service := range []string{"default", "root", "branch", "leaf"} {
		clients[service] = serviceClient(t, service)
	}
	probe := &serviceProbe{}
	root := attachServiceProbe(t, manifest, clients, probe)
	for _, tc := range []struct{ path, service string }{{"multi", "root"}, {"multi inherited", "root"}, {"multi branch inherited", "branch"}, {"multi branch leaf", "leaf"}} {
		probe.path, probe.service = strings.Fields(tc.path), tc.service
		root.SetArgs(probe.path)
		if err := root.ExecuteContext(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if probe.resolutions != 4 || probe.factories != 4 || probe.executions != 4 {
		t.Fatalf("resolution/factory/execution counts = %d/%d/%d", probe.resolutions, probe.factories, probe.executions)
	}
}

type serviceProbe struct {
	path                               []string
	service                            string
	resolutions, factories, executions int
}

func attachServiceProbe(t *testing.T, manifest pluginsdk.Manifest, clients map[string]*api.Client, probe *serviceProbe) *cobra.Command {
	t.Helper()
	root := attachRequest(t, manifest, func(_ context.Context, service string, req pluginsdk.ExecuteRequest) (*api.Client, error) {
		probe.resolutions++
		if service != probe.service || !reflect.DeepEqual(req.CommandPath, probe.path) {
			t.Errorf("resolution = %q, %v", service, req.CommandPath)
		}
		if req.Endpoint != "" || req.Context != nil {
			t.Error("host metadata injected before resolution")
		}
		return clients[service], nil
	}, func(client *http.Client) pluginsdk.Plugin {
		probe.factories++
		if client != clients[probe.service].HTTPClient() {
			t.Error("wrong service transport injected")
		}
		return &fakePlugin{execute: func(ctx context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			probe.executions++
			checkServiceRequest(t, req, probe.service, clients[probe.service])
			publicClient, err := httpclient.New(req.Endpoint, client)
			if err != nil {
				return pluginsdk.ExecuteResponse{}, err
			}
			data, err := publicClient.Do(ctx, http.MethodGet, "/probe", nil, nil, nil)
			return pluginsdk.ExecuteResponse{Data: data}, err
		}}
	})
	return root
}

func serviceClient(t *testing.T, service string) *api.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/"+service+"/probe" || r.Method != http.MethodGet {
			t.Error("incorrect host or service route")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("service transport authentication missing")
		}
		if _, err := io.WriteString(w, `{"ok":true}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := api.New(server.URL+"/api/"+service, &http.Client{Transport: bearerTransport{base: server.Client().Transport}})
	if err != nil {
		t.Fatal(err)
	}
	return client.WithContext(map[string]string{"service": service, "stack": "fixture"})
}

func checkServiceRequest(t *testing.T, req pluginsdk.ExecuteRequest, service string, client *api.Client) {
	t.Helper()
	if req.Endpoint != client.Endpoint() || !maps.Equal(req.Context, map[string]string{"service": service, "stack": "fixture"}) {
		t.Error("resolved endpoint or metadata lost")
	}
	if strings.Contains(string(mustRequestJSON(t, req)), "test-token") {
		t.Error("authentication material entered plugin request")
	}
	req.Context["stack"] = "handler-mutation"
	if client.Context()["stack"] != "fixture" {
		t.Error("handler changed host metadata")
	}
}

func mustRequestJSON(t *testing.T, req pluginsdk.ExecuteRequest) []byte {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRequestResolverReceivesValidatedBodyAndFlags(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"inline", "file", "stdin"} {
		t.Run(kind, func(t *testing.T) { t.Parallel(); checkResolvedBody(t, kind) })
	}
}

func checkResolvedBody(t *testing.T, kind string) {
	t.Helper()
	const body = `{"amount":90071992547409930001}`
	client := serviceClient(t, "ledger")
	resolutions, executions := 0, 0
	root := attachRequest(t, testManifest(), func(_ context.Context, service string, req pluginsdk.ExecuteRequest) (*api.Client, error) {
		resolutions++
		if service != "ledger" || req.Endpoint != "" || req.Context != nil {
			t.Error("invalid resolution boundary")
		}
		checkResolvedFlags(t, req, body)
		return client, nil
	}, func(injected *http.Client) pluginsdk.Plugin {
		if injected != client.HTTPClient() {
			t.Error("injected transport changed")
		}
		return &fakePlugin{execute: func(_ context.Context, req pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
			executions++
			checkResolvedFlags(t, req, body)
			if req.Endpoint != client.Endpoint() || !maps.Equal(req.Context, client.Context()) {
				t.Error("missing host endpoint/context")
			}
			req.Flags["tenant"] = "handler-mutation"
			return pluginsdk.ExecuteResponse{Data: req.Body}, nil
		}}
	})
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetIn(strings.NewReader(body))
	root.SetArgs([]string{"--connection=private-host", "ledger", "write", "fixture", "--tenant=tenant-1", "--confirm", "--data=" + bodyInput(t, kind, body)})
	if err := root.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, output.Bytes()); err != nil {
		t.Fatal(err)
	}
	if resolutions != 1 || executions != 1 || compact.String() != body {
		t.Fatalf("calls/output = %d/%d/%s", resolutions, executions, output.String())
	}
	checkCallerTenant(t, root)
}

func checkCallerTenant(t *testing.T, root *cobra.Command) {
	t.Helper()
	cmd, _, err := root.Find([]string{"ledger", "write"})
	if err != nil {
		t.Fatal(err)
	}
	tenant, err := cmd.InheritedFlags().GetString("tenant")
	if err != nil || tenant != "tenant-1" {
		t.Fatalf("handler changed caller flag: %q, %v", tenant, err)
	}
}

func checkResolvedFlags(t *testing.T, req pluginsdk.ExecuteRequest, body string) {
	t.Helper()
	if string(req.Body) != body || req.Flags["limit"] != "10" || req.Flags["enabled"] != "true" || req.Flags["tenant"] != "tenant-1" || req.Flags["confirm"] != "true" {
		t.Error("body precision or validated defaults lost")
	}
	if !req.ChangedFlags["data"] || !req.ChangedFlags["confirm"] || req.ChangedFlags["limit"] {
		t.Error("changed flag provenance lost")
	}
	if _, found := req.Flags["connection"]; found {
		t.Error("host flags entered plugin request")
	}
	if !reflect.DeepEqual(req.Args, []string{"fixture"}) || !reflect.DeepEqual(req.CommandPath, []string{"ledger", "write"}) {
		t.Error("validated route or args lost")
	}
}
