package plugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/interactive"
	"github.com/formancehq/fctl/v4/internal/plugin"
	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

const interactionBody = `{"postings":[{"amount":900719925474099312345678901234567890}],"metadata":{"exact":18446744073709551616}}`

type interactionAnswer struct {
	title, kind, value string
	options            []interactive.Option
}

// Match the schema, independent of whether the host groups static fields.
type interactionRunner struct {
	t       *testing.T
	answers []interactionAnswer
	next    int
	err     error
	fields  int
}

func (r *interactionRunner) Run(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
	r.t.Helper()
	values := make([]string, len(fields))
	for i, field := range fields {
		r.fields++
		if r.next >= len(r.answers) {
			r.t.Fatalf("unexpected form field %q (%s)", field.Title, field.Kind)
		}
		answer := r.answers[r.next]
		r.next++
		if field.Title != answer.title || field.Kind != answer.kind {
			r.t.Fatalf("field = %q (%s), want %q (%s)", field.Title, field.Kind, answer.title, answer.kind)
		}
		if answer.options != nil && !reflect.DeepEqual(field.Options, answer.options) {
			r.t.Fatalf("choices = %+v, want %+v", field.Options, answer.options)
		}
		values[i] = answer.value
	}
	if r.err != nil {
		return nil, r.err
	}
	return values, nil
}

type interactionHTTPCall struct {
	method, path string
	query        url.Values
	body         []byte
}

type interactionFixture struct {
	t                  *testing.T
	manifest           pluginsdk.Manifest
	server             *httptest.Server
	metadata           map[string]string
	resolutions        int
	input, out, stderr bytes.Buffer
	mu                 sync.Mutex
	requests           []pluginsdk.ExecuteRequest
	calls              []interactionHTTPCall
	list               func(*http.Request) string
	onResolve          func()
}

type interactionPlugin struct {
	fixture *interactionFixture
	client  *http.Client
}

func (p *interactionPlugin) GetManifest(context.Context) (pluginsdk.Manifest, error) {
	return p.fixture.manifest, nil
}

func (p *interactionPlugin) Execute(ctx context.Context, request pluginsdk.ExecuteRequest) (pluginsdk.ExecuteResponse, error) {
	request, err := pluginsdk.NormalizeRequest(p.fixture.manifest, request)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	copyRequest := request
	copyRequest.Flags = maps.Clone(request.Flags)
	copyRequest.ChangedFlags = maps.Clone(request.ChangedFlags)
	copyRequest.Context = maps.Clone(request.Context)
	copyRequest.Body = slices.Clone(request.Body)
	p.fixture.mu.Lock()
	p.fixture.requests = append(p.fixture.requests, copyRequest)
	p.fixture.mu.Unlock()
	client, err := api.New(request.Endpoint, p.client)
	if err != nil {
		return pluginsdk.ExecuteResponse{}, err
	}
	method, path := http.MethodGet, "/choices"
	switch request.CommandPath[len(request.CommandPath)-1] {
	case "create":
		method, path = http.MethodPost, "/ledgers"
	case "delete":
		method, path = http.MethodDelete, api.Path("ledgers", request.Args[0])
	case "post-choices":
		method = http.MethodPost
	}
	query := url.Values{}
	for _, name := range []string{"cursor", "after", "page-size", "stack-id", "organization-id", "bucket"} {
		if value := request.Flags[name]; value != "" {
			query.Set(name, value)
		}
	}
	data, err := client.Do(ctx, method, path, query, request.Body, http.Header{"Authorization": {"Bearer fixture-auth"}})
	return pluginsdk.ExecuteResponse{Data: data}, err
}

func interactionFixtureManifest() pluginsdk.Manifest {
	source := &pluginsdk.ChoiceSource{CommandPath: []string{"ledger", "list"}, ValueField: "id", LabelFields: []string{"name"}, Flags: map[string]string{"stack-id": "$stack"}}
	return pluginsdk.Manifest{Name: "ledger", Version: "test", Service: "ledger", ProtocolVersion: pluginsdk.ProtocolVersion, Root: pluginsdk.CommandSpec{
		Use: "ledger", Subcommands: []pluginsdk.CommandSpec{
			{Use: "create NAME", Runnable: true, Args: pluginsdk.ArgsSpec{Min: 1, Max: 1}, Flags: []pluginsdk.FlagSpec{
				{Name: "bucket", Type: "string", Required: true},
				{Name: "data", Type: "string", Body: true, Required: true},
			}, Inputs: []pluginsdk.InputSpec{
				{Title: "Bucket", Kind: "select", Flag: "bucket", Required: true, Source: source},
				{Title: "Ledger name", Kind: "input", Argument: new(0), Required: true},
				{Title: "Amount", Kind: "input", BodyPointer: "/postings/0/amount", ValueType: "number", Required: true},
			}},
			{Use: "list", Runnable: true, Flags: []pluginsdk.FlagSpec{
				{Name: "cursor", Type: "string"}, {Name: "stack-id", Type: "string"}, {Name: "organization-id", Type: "string"},
				{Name: "after", Type: "string"}, {Name: "page-size", Type: "uint32", Default: "10"},
			}},
			{Use: "delete NAME", Runnable: true, Args: pluginsdk.ArgsSpec{Min: 1, Max: 1}, Confirm: true, Flags: []pluginsdk.FlagSpec{{Name: "confirm", Type: "bool", Default: "false"}}, Inputs: []pluginsdk.InputSpec{
				{Title: "Ledger", Kind: "select", Argument: new(0), Required: true, Source: source},
			}},
			{Use: "post-choices", Runnable: true},
		},
	}}
}

func newInteractionFixture(t *testing.T) *interactionFixture {
	t.Helper()
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	f := &interactionFixture{t: t, manifest: interactionFixtureManifest(), metadata: map[string]string{"stack": "stack-1"}}
	f.list = func(*http.Request) string { return `{"data":[{"id":"bucket-main","name":"Main"}],"hasMore":false}` }
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20+1))
		if err != nil {
			t.Error(err)
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer fixture-auth" {
			t.Error("plugin request lost its authenticated HTTP client")
		}
		f.mu.Lock()
		f.calls = append(f.calls, interactionHTTPCall{method: r.Method, path: r.URL.EscapedPath(), query: r.URL.Query(), body: body})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			writeInteractionJSON(t, w, f.list(r))
			return
		}
		writeInteractionJSON(t, w, `{"ok":true}`)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func writeInteractionJSON(t *testing.T, w io.Writer, data string) {
	t.Helper()
	if _, err := io.WriteString(w, data); err != nil {
		t.Error(err)
	}
}

// An isolated host root with a real registry and adapter; no product imports.
func (f *interactionFixture) execRoot(runner interactive.Runner, args ...string) error {
	registry := new(plugin.Registry)
	factory := func(client *http.Client) pluginsdk.Plugin { return &interactionPlugin{fixture: f, client: client} }
	if err := registry.Register(f.t.Context(), factory(nil), factory); err != nil {
		f.t.Fatal(err)
	}
	registry.Freeze()
	root := &cobra.Command{Use: "fctl", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().Bool("no-input", false, "disable forms")
	root.PersistentFlags().String("color", "never", "terminal color")
	root.SetIn(&f.input)
	root.SetOut(&f.out)
	root.SetErr(&f.stderr)
	adapter := plugin.NewCommand(registry, func(_ context.Context, service string) (*api.Client, error) {
		f.resolutions++
		if f.onResolve != nil {
			f.onResolve()
		}
		if service != f.manifest.Service {
			f.t.Fatalf("resolved unexpected service %q", service)
		}
		client := f.server.Client()
		client.Timeout = time.Second
		resolved, err := api.New(f.server.URL, client)
		if err != nil {
			return nil, err
		}
		return resolved.WithContext(f.metadata), nil
	})
	if err := adapter.AddTo(root); err != nil {
		f.t.Fatal(err)
	}
	ctx := f.t.Context()
	if runner != nil {
		ctx = interactive.WithRunner(ctx, runner)
	}
	root.SetContext(ctx)
	root.SetArgs(args)
	return root.Execute()
}

func (f *interactionFixture) httpCalls() []interactionHTTPCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

func (f *interactionFixture) assertWrites(want int) {
	f.t.Helper()
	writes := 0
	for _, call := range f.httpCalls() {
		if call.method != http.MethodGet {
			writes++
		}
	}
	if writes != want {
		f.t.Fatalf("HTTP writes = %d, want %d", writes, want)
	}
}

func TestInteractionCanceledLedgerCreateDoesNotWrite(t *testing.T) {
	f := newInteractionFixture(t)
	runner := &interactionRunner{t: t, answers: []interactionAnswer{
		{title: "Bucket", kind: "select", value: "bucket-main"},
		{title: "Ledger name", kind: "input"},
		{title: "Amount", kind: "input"},
	}, err: interactive.ErrCanceled}
	err := f.execRoot(runner, "ledger", "create")
	if !errors.Is(err, interactive.ErrCanceled) {
		t.Fatalf("create error = %v, want canceled", err)
	}
	f.assertWrites(0)
	if calls := f.httpCalls(); len(calls) != 1 || calls[0].method != http.MethodGet {
		t.Fatalf("cancellation should permit only discovery; got %+v", calls)
	}
	if f.out.Len() != 0 {
		t.Fatal("canceled creation printed a successful result")
	}
}

func TestInteractionBindsAllInputKindsWithoutRoundingNumbers(t *testing.T) {
	f := newInteractionFixture(t)
	create := &f.manifest.Root.Subcommands[0]
	create.Flags[1].Default = `{"postings":[{"asset":"USD/2","existing":9007199254740993123456789}],"metadata":{"keep":18446744073709551616}}`
	create.Inputs = append([]pluginsdk.InputSpec{{Title: "Organization", Kind: "select", Context: "organization", Required: true, Options: []pluginsdk.InputOption{{Label: "Example", Value: "org-new"}}}}, create.Inputs...)
	create.Inputs[1].Source.Flags["organization-id"] = "$organization"
	create.Inputs = append(create.Inputs, pluginsdk.InputSpec{Title: "Metadata id", Kind: "input", BodyPointer: "/metadata/a~1b/~0id", ValueType: "number", Required: true})
	runner := &interactionRunner{t: t, answers: []interactionAnswer{
		{title: "Organization", kind: "select", value: "org-new"},
		{title: "Bucket", kind: "select", value: "bucket-main"},
		{title: "Ledger name", kind: "input", value: "demo"},
		{title: "Amount", kind: "input", value: "900719925474099312345678901234567890"},
		{title: "Metadata id", kind: "input", value: "18446744073709551617"},
	}}
	if err := f.execRoot(runner, "ledger", "create"); err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	calls := f.httpCalls()
	if len(calls) != 2 || calls[0].query.Get("organization-id") != "org-new" || calls[0].query.Get("stack-id") != "stack-1" {
		t.Fatalf("choice source did not receive previously resolved context: %+v", calls)
	}
	const wantBody = `{"metadata":{"a/b":{"~id":18446744073709551617},"keep":18446744073709551616},"postings":[{"amount":900719925474099312345678901234567890,"asset":"USD/2","existing":9007199254740993123456789}]}`
	if string(calls[1].body) != wantBody {
		t.Fatal("body pointer binding rounded numbers or lost existing JSON")
	}
	request := f.requests[len(f.requests)-1]
	if !reflect.DeepEqual(request.Args, []string{"demo"}) || request.Flags["bucket"] != "bucket-main" || !request.ChangedFlags["bucket"] || request.Context["organization"] != "org-new" || string(request.Body) != wantBody {
		t.Fatal("interactive values did not survive the normalized SDK request")
	}
	if runner.next != len(runner.answers) || f.out.String() != "{\n  \"ok\": true\n}\n" {
		t.Fatal("wizard did not collect all fields and print the service response")
	}
}

func TestInteractionExplicitInputsSkipEveryForm(t *testing.T) {
	for _, kind := range []string{"inline", "file", "stdin"} {
		t.Run(kind, func(t *testing.T) { testInteractionExplicitInputs(t, kind) })
	}
}

func testInteractionExplicitInputs(t *testing.T, kind string) {
	t.Helper()
	f := newInteractionFixture(t)
	input := f.bodyArgument(kind)
	runner := &interactionRunner{t: t}
	if err := f.execRoot(runner, "ledger", "create", "demo", "--bucket", "bucket-main", "--data", input); err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	calls := f.httpCalls()
	if len(calls) != 1 || calls[0].method != http.MethodPost || string(calls[0].body) != interactionBody || runner.fields != 0 {
		t.Fatal("explicit input prompted, discovered choices, or changed raw JSON")
	}
}

func (f *interactionFixture) bodyArgument(kind string) string {
	f.t.Helper()
	switch kind {
	case "file":
		path := filepath.Join(f.t.TempDir(), "request.json")
		if err := os.WriteFile(path, []byte(interactionBody), 0o600); err != nil {
			f.t.Fatal(err)
		}
		return "@" + path
	case "stdin":
		if _, err := f.input.WriteString(interactionBody); err != nil {
			f.t.Fatal(err)
		}
		return "-"
	default:
		return interactionBody
	}
}

func TestInteractionDisabledMissingInputsFailBeforeAuthentication(t *testing.T) {
	for _, mode := range []string{"no-input", "CI", "FCTL_NO_INPUT", "nonTTY"} {
		for _, tc := range []struct {
			name string
			args []string
		}{
			{"argument", []string{"ledger", "create", "--bucket", "bucket-main", "--data", interactionBody}},
			{"flag", []string{"ledger", "create", "demo", "--data", interactionBody}},
			{"body", []string{"ledger", "create", "demo", "--bucket", "bucket-main"}},
			{"confirmation", []string{"ledger", "delete", "demo"}},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				testInteractionMissingInput(t, mode, tc.args)
			})
		}
	}
}

func testInteractionMissingInput(t *testing.T, mode string, args []string) {
	t.Helper()
	f := newInteractionFixture(t)
	runner := &interactionRunner{t: t}
	injected, prefix := disabledInteractionRunner(t, mode, runner)
	if err := f.execRoot(injected, append(prefix, args...)...); err == nil {
		t.Fatal("missing input was accepted with forms disabled")
	}
	if f.resolutions != 0 || runner.fields != 0 || len(f.httpCalls()) != 0 || len(f.requests) != 0 {
		t.Fatal("missing input invoked authentication, discovery, a form, or the plugin")
	}
}

func disabledInteractionRunner(t *testing.T, mode string, runner interactive.Runner) (interactive.Runner, []string) {
	t.Helper()
	switch mode {
	case "no-input":
		return runner, []string{"--no-input"}
	case "CI":
		t.Setenv("CI", "true")
	case "FCTL_NO_INPUT":
		t.Setenv("FCTL_NO_INPUT", "true")
	case "nonTTY":
		return nil, nil
	}
	return runner, nil
}

func TestInteractionChoicesPaginateAndDeduplicate(t *testing.T) {
	f := newInteractionFixture(t)
	f.list = func(r *http.Request) string {
		switch r.URL.Query().Get("cursor") {
		case "":
			return `{"cursor":{"data":[{"id":"first","name":"First"}],"hasMore":true,"next":"page-2"}}`
		case "page-2":
			return `{"cursor":{"data":[{"id":"first","name":"Duplicate"},{"id":9007199254740993123456789,"name":"Large\nledger"}],"hasMore":true,"next":"page-3"}}`
		case "page-3":
			return `{"cursor":{"data":[{"id":"first"}],"hasMore":false}}`
		default:
			t.Error("unexpected pagination cursor")
			return `{"data":[]}`
		}
	}
	runner := &interactionRunner{t: t, answers: []interactionAnswer{{title: "Bucket", kind: "select", value: "9007199254740993123456789", options: []interactive.Option{
		{Label: "First (first)", Value: "first"},
		{Label: `Large\u000aledger (9007199254740993123456789)`, Value: "9007199254740993123456789"},
	}}}}
	if err := f.execRoot(runner, "ledger", "create", "demo", "--data", interactionBody); err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	calls := f.httpCalls()
	if len(calls) != 4 {
		t.Fatalf("HTTP calls = %d, want three pages and one create", len(calls))
	}
	for i, cursor := range []string{"", "page-2", "page-3"} {
		if calls[i].method != http.MethodGet || calls[i].query.Get("cursor") != cursor || calls[i].query.Get("stack-id") != "stack-1" {
			t.Fatalf("discovery page %d lost cursor or host context", i)
		}
	}
	if calls[3].query.Get("bucket") != "9007199254740993123456789" {
		t.Fatal("selected numeric identifier lost precision before creation")
	}
}

func TestInteractionInvalidChoiceDiscoveryCannotWrite(t *testing.T) {
	for _, tc := range []struct {
		name, response, errorText string
		getCalls                  int
	}{
		{"repeated cursor", `{"data":[{"id":"first"}],"hasMore":true,"next":"same"}`, "cannot enumerate choices", 2},
		{"missing next cursor", `{"data":[{"id":"first"}],"hasMore":true}`, "cannot enumerate choices", 1},
		{"missing array", `{"cursor":{"hasMore":false}}`, "no data array", 1},
		{"missing identifier", `{"data":[{"name":"nameless"}]}`, "lacks id", 1},
		{"invalid resource", `{"data":["wrong"]}`, "not an object", 1},
		{"empty list", `{"data":[]}`, "no resources available", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newInteractionFixture(t)
			f.list = func(*http.Request) string { return tc.response }
			runner := &interactionRunner{t: t}
			err := f.execRoot(runner, "ledger", "create", "demo", "--data", interactionBody)
			if err == nil || !strings.Contains(err.Error(), tc.errorText) {
				t.Fatalf("discovery error = %v, want %q", err, tc.errorText)
			}
			f.assertWrites(0)
			if len(f.httpCalls()) != tc.getCalls || runner.fields != 0 {
				t.Fatal("failed discovery did not stop before forms and writes")
			}
		})
	}
}

func TestInteractionReadOnlyDiscoveryRefusesPOST(t *testing.T) {
	f := newInteractionFixture(t)
	input := &f.manifest.Root.Subcommands[0].Inputs[0]
	input.Source = &pluginsdk.ChoiceSource{CommandPath: []string{"ledger", "post-choices"}, ValueField: "id"}
	runner := &interactionRunner{t: t}
	err := f.execRoot(runner, "ledger", "create", "demo", "--data", interactionBody)
	if err == nil || !strings.Contains(err.Error(), "refused a mutating request") {
		t.Fatalf("discovery error = %v, want read-only rejection", err)
	}
	f.assertWrites(0)
	if len(f.httpCalls()) != 0 || runner.fields != 0 {
		t.Fatal("mutating choice discovery reached HTTP or the form")
	}
}

func TestInteractionConfirmationControlsDELETE(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		t.Run(fmt.Sprint(accepted), func(t *testing.T) { testInteractionDeletion(t, accepted) })
	}
}

func testInteractionDeletion(t *testing.T, accepted bool) {
	t.Helper()
	f := newInteractionFixture(t)
	runner := &interactionRunner{t: t, answers: []interactionAnswer{
		{title: "Ledger", kind: "select", value: "bucket-main"},
		{title: "Confirm ledger delete bucket-main?", kind: "confirm", value: fmt.Sprint(accepted)},
	}}
	err := f.execRoot(runner, "ledger", "delete")
	if !accepted {
		if !errors.Is(err, interactive.ErrCanceled) {
			t.Fatalf("declined delete error = %v, want canceled", err)
		}
		f.assertWrites(0)
		if f.out.Len() != 0 {
			t.Fatal("declined deletion printed success")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	calls := f.httpCalls()
	if len(calls) != 2 || calls[1].method != http.MethodDelete || calls[1].path != "/ledgers/bucket-main" {
		t.Fatal("accepted confirmation did not delete the selected resource exactly once")
	}
	request := f.requests[len(f.requests)-1]
	if request.Flags["confirm"] != "true" || !request.ChangedFlags["confirm"] {
		t.Fatal("accepted confirmation was not enforced by the SDK")
	}
}

func TestInteractionNumberInputsRejectNonNumbers(t *testing.T) {
	for _, value := range []string{`"9007199254740993"`, "true", "null", `{"amount":1}`, "[1]", "fixture-secret-not-a-number"} {
		t.Run(value, func(t *testing.T) {
			f := newInteractionFixture(t)
			runner := &interactionRunner{t: t, answers: []interactionAnswer{{title: "Amount", kind: "input", value: value}}}
			err := f.execRoot(runner, "ledger", "create", "demo", "--bucket", "bucket-main")
			if err == nil {
				t.Fatal("non-number was accepted as a JSON number")
			}
			f.assertWrites(0)
			if strings.Contains(err.Error()+f.out.String()+f.stderr.String(), value) {
				t.Fatal("validation printed a secret input")
			}
		})
	}
}

func TestInteractionAlternativeFlagSkipsPathOnlyWhenExplicit(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprint(explicit), func(t *testing.T) { testInteractionAlternativeFlag(t, explicit) })
	}
}

func testInteractionAlternativeFlag(t *testing.T, explicit bool) {
	t.Helper()
	f := newInteractionFixture(t)
	create := &f.manifest.Root.Subcommands[0]
	create.Args.Min = 0
	create.Inputs[1].AlternativeFlag = "data"
	args := []string{"ledger", "create", "--bucket", "bucket-main"}
	runner := &interactionRunner{t: t}
	if explicit {
		args = append(args, "--data", interactionBody)
	} else {
		create.Flags[1].Default = interactionBody
		runner.answers = []interactionAnswer{{title: "Ledger name", kind: "input", value: "demo"}}
	}
	if err := f.execRoot(runner, args...); err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	if calls := f.httpCalls(); len(calls) != 1 || string(calls[0].body) != interactionBody {
		t.Fatal("alternative flag changed the explicit JSON or triggered discovery")
	}
	request := f.requests[0]
	if explicit && (len(request.Args) != 0 || runner.fields != 0) {
		t.Fatal("explicit alternative flag did not bypass the path form")
	}
	if !explicit && (!reflect.DeepEqual(request.Args, []string{"demo"}) || runner.fields != 1) {
		t.Fatal("an unchanged body default incorrectly bypassed the path form")
	}
}

func TestInteractionPreferredPrefixSuggestsWithoutSelecting(t *testing.T) {
	f := newInteractionFixture(t)
	f.manifest.Root.Subcommands[0].Inputs[0].Source.PreferredPrefix = "v4."
	f.list = func(*http.Request) string {
		return `{"data":[{"id":"v3.2.0"},{"id":"v4.0.2"},{"id":"v4.0.1"},{"id":"v3.1.0"}]}`
	}
	runner := &interactionRunner{t: t, answers: []interactionAnswer{{title: "Bucket", kind: "select", value: "v3.2.0", options: []interactive.Option{
		{Label: "v4.0.2", Value: "v4.0.2"}, {Label: "v4.0.1", Value: "v4.0.1"},
		{Label: "v3.2.0", Value: "v3.2.0"}, {Label: "v3.1.0", Value: "v3.1.0"},
	}}}}
	if err := f.execRoot(runner, "ledger", "create", "demo", "--data", interactionBody); err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	calls := f.httpCalls()
	if len(calls) != 2 || runner.fields != 1 || calls[1].query.Get("bucket") != "v3.2.0" {
		t.Fatal("catalog suggestion replaced the user's explicit selection")
	}
}

func TestInteractionBodyFileClosesBeforeAuthAndOnValidationFailure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("file descriptor inspection requires Linux /proc/self/fd; macOS /dev/fd does not expose matching file inodes")
	}
	for _, valid := range []bool{false, true} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) { testInteractionBodyFile(t, valid) })
	}
}

func testInteractionBodyFile(t *testing.T, valid bool) {
	t.Helper()
	f := newInteractionFixture(t)
	file, err := os.CreateTemp(t.TempDir(), "body-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeInteractionFile(t, file) })
	path := file.Name()
	body := "{ invalid JSON"
	if valid {
		body = interactionBody
	}
	if _, err := file.WriteString(body); err != nil {
		t.Fatal(err)
	}
	if !interactionFileOpen(t, path) {
		t.Fatal("descriptor inspection cannot detect an open body file")
	}
	closeInteractionFile(t, file)
	f.onResolve = func() { assertInteractionFileClosed(t, path) }
	err = f.execRoot(&interactionRunner{t: t}, "ledger", "create", "demo", "--bucket", "bucket-main", "--data", "@"+path)
	assertInteractionFileClosed(t, path)
	if valid {
		if err != nil {
			t.Fatal(err)
		}
		f.assertWrites(1)
		return
	}
	if err == nil || f.resolutions != 0 {
		t.Fatal("invalid body reached authentication")
	}
	f.assertWrites(0)
}

func assertInteractionFileClosed(t *testing.T, path string) {
	t.Helper()
	if interactionFileOpen(t, path) {
		t.Fatal("request body file is still open")
	}
}

func interactionFileOpen(t *testing.T, path string) bool {
	t.Helper()
	want, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range interactionDescriptorNames(t) {
		info, err := os.Stat(filepath.Join("/proc/self/fd", name))
		if err == nil && os.SameFile(info, want) {
			return true
		}
	}
	return false
}

func TestInteractionExplicitFalseConfirmationFailsBeforeAuth(t *testing.T) {
	f := newInteractionFixture(t)
	runner := &interactionRunner{t: t}
	err := f.execRoot(runner, "ledger", "delete", "demo", "--confirm=false")
	if err == nil || !strings.Contains(err.Error(), "requires --confirm") {
		t.Fatalf("explicit false confirmation error = %v", err)
	}
	f.assertWrites(0)
	if f.resolutions != 0 || runner.fields != 0 || len(f.httpCalls()) != 0 || len(f.requests) != 0 {
		t.Fatal("explicitly declined deletion reached authentication, discovery, a form, or the plugin")
	}
}

func interactionDescriptorNames(t *testing.T) []string {
	t.Helper()
	dir, err := os.Open("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	names, readErr := dir.Readdirnames(-1)
	if err := errors.Join(readErr, dir.Close()); err != nil {
		t.Fatal(err)
	}
	return names
}

func closeInteractionFile(t *testing.T, file *os.File) {
	t.Helper()
	if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}
}

func TestInteractionSecretStringNeverAppearsInOutput(t *testing.T) {
	f := newInteractionFixture(t)
	create := &f.manifest.Root.Subcommands[0]
	create.Flags = append(create.Flags, pluginsdk.FlagSpec{Name: "credential", Type: "string", Required: true})
	create.Inputs = append(create.Inputs, pluginsdk.InputSpec{Title: "Credential", Kind: "input", Flag: "credential", Required: true, Secret: true})
	const secret = "fixture-secret-do-not-print"
	runner := &interactionRunner{t: t, answers: []interactionAnswer{{title: "Credential", kind: "input", value: secret}}}
	if err := f.execRoot(runner, "ledger", "create", "demo", "--bucket", "bucket-main", "--data", interactionBody); err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	if f.requests[0].Flags["credential"] != secret || !f.requests[0].ChangedFlags["credential"] {
		t.Fatal("secret value did not reach the plugin")
	}
	if strings.Contains(f.out.String()+f.stderr.String(), secret) {
		t.Fatal("command printed a secret string")
	}
}

func TestInteractionAfterFieldLoadsAllRowsWithExactIDs(t *testing.T) {
	f := newInteractionFixture(t)
	source := f.manifest.Root.Subcommands[0].Inputs[0].Source
	source.AfterField = "id"
	source.Flags["page-size"] = "2"
	firstPage, options := interactionKeysetPage(t)
	const lastID = "9007199254740993099"
	const selectedID = "18446744073709551617"
	options = append(options, interactive.Option{Label: selectedID, Value: selectedID})
	f.list = func(r *http.Request) string {
		if r.URL.Query().Get("after") == lastID {
			return `{"data":[{"id":18446744073709551617}]}`
		}
		return firstPage
	}
	runner := &interactionRunner{t: t, answers: []interactionAnswer{{title: "Bucket", kind: "select", value: selectedID, options: options}}}
	if err := f.execRoot(runner, "ledger", "create", "demo", "--data", interactionBody); err != nil {
		t.Fatal(err)
	}
	f.assertWrites(1)
	calls := f.httpCalls()
	if len(calls) != 3 || calls[0].query.Get("after") != "" || calls[1].query.Get("after") != lastID || calls[2].query.Get("bucket") != selectedID {
		t.Fatal("keyset pagination lost its exact after value or the final selection")
	}
	for _, call := range calls[:2] {
		if call.method != http.MethodGet || call.query.Get("page-size") != "100" || call.query.Has("cursor") {
			t.Fatal("keyset discovery did not use GET, page-size 100, and --after")
		}
	}
}

func TestInteractionAfterFieldStopsOnRepeatedPage(t *testing.T) {
	f := newInteractionFixture(t)
	f.manifest.Root.Subcommands[0].Inputs[0].Source.AfterField = "id"
	page, _ := interactionKeysetPage(t)
	f.list = func(*http.Request) string { return page }
	runner := &interactionRunner{t: t}
	err := f.execRoot(runner, "ledger", "create", "demo", "--data", interactionBody)
	if err == nil || !strings.Contains(err.Error(), "cannot enumerate choices") {
		t.Fatalf("repeated keyset page error = %v", err)
	}
	f.assertWrites(0)
	if len(f.httpCalls()) != 2 || runner.fields != 0 {
		t.Fatal("repeated keyset page continued enumerating or invoked the form")
	}
}

func interactionKeysetPage(t *testing.T) (string, []interactive.Option) {
	t.Helper()
	rows := make([]map[string]json.Number, 100)
	options := make([]interactive.Option, 100)
	for i := range 100 {
		id := fmt.Sprintf("9007199254740993%03d", i)
		rows[i] = map[string]json.Number{"id": json.Number(id)}
		options[i] = interactive.Option{Label: id, Value: id}
	}
	page, err := json.Marshal(map[string]any{"data": rows})
	if err != nil {
		t.Fatal(err)
	}
	return string(page), options
}

type confirmationProbeRunner struct {
	t      *testing.T
	hidden []string
	titles []string
}

func (r *confirmationProbeRunner) Run(_ context.Context, cmd *cobra.Command, fields []interactive.Field) ([]string, error) {
	r.t.Helper()
	values := make([]string, len(fields))
	for i, field := range fields {
		if field.Kind != "confirm" || field.Default != "false" {
			r.t.Fatal("fully supplied command did not reach a confirmation with a safe default")
		}
		assertConfirmationHidden(r.t, field.Title+field.Description, r.hidden)
		r.titles = append(r.titles, field.Title)
		// Exercise the host's rendered confirmation text and stderr boundary.
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), field.Title); err != nil {
			return nil, err
		}
		values[i] = "false"
	}
	return values, nil
}

func TestInteractionConfirmationHidesExplicitPositionalSecrets(t *testing.T) {
	for _, tc := range []struct {
		name        string
		secretIndex int
		alternative bool
	}{
		{name: "first argument", secretIndex: 0},
		{name: "second argument", secretIndex: 1},
		{name: "alternative argument", secretIndex: 1, alternative: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testInteractionPositionalSecret(t, tc.secretIndex, tc.alternative)
		})
	}
}

func testInteractionPositionalSecret(t *testing.T, secretIndex int, alternative bool) {
	t.Helper()
	f := newInteractionFixture(t)
	const secret = "fixture-plaintext-positional-secret"
	args := []string{"app-main", "app-main"}
	args[secretIndex] = secret
	secretInput := pluginsdk.InputSpec{Title: "Credential", Kind: "input", Argument: new(secretIndex), Required: true, Secret: true}
	flags := []pluginsdk.FlagSpec{{Name: "confirm", Type: "bool", Default: "false"}}
	if alternative {
		secretInput.Argument = nil
		secretInput.Flag, secretInput.AlternativeArgument = "credential", new(secretIndex)
		flags = append(flags, pluginsdk.FlagSpec{Name: "credential", Type: "string"})
	}
	f.manifest.Root.Subcommands = []pluginsdk.CommandSpec{{
		Use: "delete APP CREDENTIAL", Runnable: true, Confirm: true, Args: pluginsdk.ArgsSpec{Min: 2, Max: 2}, Flags: flags,
		Inputs: []pluginsdk.InputSpec{
			{Title: "Application", Kind: "input", Argument: new(1 - secretIndex), Required: true},
			secretInput,
		},
	}}
	runner := &confirmationProbeRunner{t: t, hidden: []string{secret}}
	err := f.execRoot(runner, append([]string{"ledger", "delete"}, args...)...)
	assertCanceledConfirmation(f, runner, err, []string{"app-main"})
}

func TestInteractionVariableDeleteConfirmationNamesAppAndVariable(t *testing.T) {
	for _, positional := range []bool{false, true} {
		t.Run(fmt.Sprint(positional), func(t *testing.T) {
			testInteractionVariableConfirmation(t, positional)
		})
	}
}

func testInteractionVariableConfirmation(t *testing.T, positional bool) {
	t.Helper()
	f := newInteractionFixture(t)
	const body = `{"metadata":{"reason":"private-body-note"}}`
	const secret = "fixture-private-flag-secret"
	variable := pluginsdk.InputSpec{Title: "Variable", Kind: "input", Flag: "key", Required: true}
	args := []string{"auth", "apps", "variables", "delete", "--app-id", "app-main", "--data", body, "--credential", secret}
	if positional {
		variable.AlternativeArgument = new(0)
		args = append(args, "DATABASE_URL")
	} else {
		args = append(args, "--key", "DATABASE_URL")
	}
	deletion := pluginsdk.CommandSpec{
		Use: "delete [KEY]", Runnable: true, Confirm: true, Args: pluginsdk.ArgsSpec{Max: 1},
		Flags: []pluginsdk.FlagSpec{
			{Name: "confirm", Type: "bool", Default: "false"},
			{Name: "app-id", Type: "string", Required: true}, {Name: "key", Type: "string"},
			{Name: "data", Type: "string", Body: true}, {Name: "credential", Type: "string"},
		},
		Inputs: []pluginsdk.InputSpec{
			{Title: "Application", Kind: "input", Flag: "app-id", Required: true},
			variable,
			{Title: "Organization", Kind: "input", Context: "organization"},
			{Title: "Stack", Kind: "input", Context: "stack"},
			{Title: "Request JSON", Kind: "text", Flag: "data", ValueType: "json"},
			{Title: "Reason", Kind: "input", BodyPointer: "/metadata/reason"},
			{Title: "Credential", Kind: "input", Flag: "credential", Secret: true},
		},
	}
	f.manifest.Name, f.manifest.Service, f.manifest.Root.Use = "auth", "auth", "auth"
	f.manifest.Root.Subcommands = []pluginsdk.CommandSpec{{Use: "apps", Subcommands: []pluginsdk.CommandSpec{
		{Use: "variables", Subcommands: []pluginsdk.CommandSpec{deletion}},
	}}}
	f.metadata["organization"] = "app-main" // The same value must appear only once.
	runner := &confirmationProbeRunner{t: t, hidden: []string{secret, body, "private-body-note"}}
	err := f.execRoot(runner, args...)
	assertCanceledConfirmation(f, runner, err, []string{"app-main", "DATABASE_URL", "stack-1"})
}

func assertCanceledConfirmation(f *interactionFixture, runner *confirmationProbeRunner, err error, resources []string) {
	f.t.Helper()
	if !errors.Is(err, interactive.ErrCanceled) {
		f.t.Fatal("declining the confirmation did not cancel the command")
	}
	f.assertWrites(0)
	if len(runner.titles) != 1 || len(f.httpCalls()) != 0 || len(f.requests) != 0 {
		f.t.Fatal("confirmation prompted more than once or reached HTTP execution")
	}
	if f.out.Len() != 0 || !strings.Contains(f.stderr.String(), runner.titles[0]) {
		f.t.Fatal("confirmation did not use stderr or printed a command result")
	}
	for _, resource := range resources {
		if strings.Count(runner.titles[0], resource) != 1 {
			f.t.Fatalf("confirmation must identify resource %q exactly once", resource)
		}
	}
	assertConfirmationHidden(f.t, f.out.String()+f.stderr.String()+strings.Join(runner.titles, "\n")+err.Error(), runner.hidden)
}

func assertConfirmationHidden(t *testing.T, text string, hidden []string) {
	t.Helper()
	for _, value := range hidden {
		if strings.Contains(text, value) {
			t.Fatal("confirmation exposed a secret or request body value")
		}
	}
}
