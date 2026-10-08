package connection

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/interactive"
)

func TestOrganizationChoicesUseVerifiedIDsAndNames(t *testing.T) {
	t.Parallel()
	choices := organizationChoices([]string{"org-b", "org-a", "org-c"}, []cloudChoiceResource{
		{ID: "org-a", Name: "Books"}, {ID: "org-b", Name: "Accounts"}, {ID: "ungranted", Name: "Other"},
	})
	want := []interactive.Option{
		{Label: "Accounts (org-b)", Value: "org-b"}, {Label: "Books (org-a)", Value: "org-a"}, {Label: "org-c", Value: "org-c"},
	}
	if !reflect.DeepEqual(choices, want) {
		t.Fatalf("choices = %+v, want %+v", choices, want)
	}
}

func TestReadyStackChoicesFilterAvailabilityAndAuthorization(t *testing.T) {
	t.Parallel()
	allowed := []string{"a", "b", "disabled", "deleted", "progressing", "wrong-org", "unknown-state"}
	resources := []cloudChoiceResource{
		{ID: "b", Name: "Books", Status: "READY", State: "ACTIVE", OrganizationID: "org"},
		{ID: "a", Name: "Books", Status: "READY"},
		{ID: "disabled", Status: "READY", State: "DISABLED"},
		{ID: "deleted", Status: "READY", State: "DELETED"},
		{ID: "progressing", Status: "PROGRESSING", State: "ACTIVE"},
		{ID: "wrong-org", Status: "READY", OrganizationID: "other"},
		{ID: "unknown-state", Status: "READY", State: "RESTORING"},
		{ID: "ungranted", Name: "Unverified", Status: "READY", State: "ACTIVE"},
		{ID: "a", Name: "Duplicate", Status: "READY"},
	}
	want := []interactive.Option{
		{Label: "Books (a) · READY", Value: "a"}, {Label: "Books (b) · READY", Value: "b"},
	}
	if choices := readyStackChoices("org", allowed, resources); !reflect.DeepEqual(choices, want) {
		t.Fatalf("choices = %+v, want %+v", choices, want)
	}
}

func TestOrganizationsForStackPreservesExplicitStack(t *testing.T) {
	t.Parallel()
	identity := cloud.IdentityInfo{
		Organizations: []string{"empty", "first", "second"},
		Stacks:        map[string][]string{"empty": {}, "first": {"one", "shared"}, "second": {"two", "shared"}},
	}
	for _, tc := range []struct {
		stack string
		want  []string
	}{
		{"", []string{"first", "second"}}, {"one", []string{"first"}},
		{"shared", []string{"first", "second"}}, {"unknown", nil},
	} {
		t.Run(tc.stack, func(t *testing.T) {
			if got := organizationsForStack(identity, tc.stack); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("organizations = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCloudCommandTargetUsesInheritedAnnotation(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "arbitrary", Annotations: map[string]string{"fctl.target": "organization"}}
	child := &cobra.Command{Use: "renamed"}
	leaf := &cobra.Command{Use: "read", Annotations: map[string]string{"fctl.target": "identity"}}
	root.AddCommand(child)
	child.AddCommand(leaf)
	if cloudCommandTarget(child) != "organization" || cloudCommandTarget(leaf) != "identity" {
		t.Fatal("target metadata was not inherited or overridden")
	}
	if cloudCommandTarget(&cobra.Command{Use: "cloud stack list"}) != "" {
		t.Fatal("command name inferred a target without metadata")
	}
}

func selectionCommand(t *testing.T, settings *Settings) (*cobra.Command, *cobra.Command) {
	t.Helper()
	root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
	settings.Bind(root)
	root.PersistentFlags().Bool("no-input", false, "Disable prompts")
	root.SetIn(strings.NewReader(""))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetContext(t.Context())
	child := &cobra.Command{Use: "command"}
	child.SetContext(t.Context())
	root.AddCommand(child)
	if err := child.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	return root, child
}

func TestApplyCloudTargetSetsInheritedFlagsAndPreservesStore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entry := NewEntry(Options{AuthMode: "cloud"})
	if err := Save(dir, Store{Active: "work", Connections: map[string]Entry{"work": entry}}); err != nil {
		t.Fatal(err)
	}
	s := &Settings{}
	root, child := selectionCommand(t, s)
	if err := root.PersistentFlags().Set("config-dir", dir); err != nil {
		t.Fatal(err)
	}
	options, err := applyCloudTarget(child, entry.Options, "org", "stack")
	if err != nil {
		t.Fatal(err)
	}
	if options.Organization != "org" || options.Stack != "stack" || !root.PersistentFlags().Changed("organization") || !root.PersistentFlags().Changed("stack") {
		t.Fatal("selected target did not update inherited flags")
	}
	resolved, original, _, _, err := s.Resolve(child)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != options || !reflect.DeepEqual(original, entry) {
		t.Fatal("selection did not resolve through the same profile or changed saved defaults")
	}
	store, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.Connections["work"], entry) {
		t.Fatal("selection persisted a target or changed the profile revision")
	}
}

func TestApplyCloudTargetNeverOverridesExplicitIDs(t *testing.T) {
	t.Parallel()
	s := &Settings{}
	root, child := selectionCommand(t, s)
	options := Options{AuthMode: "cloud", Organization: "explicit-org", Stack: "explicit-stack"}
	selected, err := applyCloudTarget(child, options, "another-org", "another-stack")
	if err != nil {
		t.Fatal(err)
	}
	if selected != options || root.PersistentFlags().Changed("organization") || root.PersistentFlags().Changed("stack") {
		t.Fatal("picker overrode explicit target IDs")
	}
}

func TestSelectCloudTargetWithoutInputDoesNotListResources(t *testing.T) {
	t.Parallel()
	for _, noInput := range []bool{false, true} {
		t.Run(map[bool]string{false: "nonTTY", true: "no-input"}[noInput], func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			t.Cleanup(server.Close)
			s := &Settings{}
			root, child := selectionCommand(t, s)
			if noInput {
				if err := root.PersistentFlags().Set("no-input", "true"); err != nil {
					t.Fatal(err)
				}
			}
			options := Options{AuthMode: "cloud", Issuer: server.URL}
			entry := NewEntry(options)
			entry.Session = &cloud.Session{IDToken: "not-a-token"}
			got, err := s.selectCloudTarget(t.Context(), child, server.Client(), options, entry, "work", t.TempDir())
			if err != nil || got != options || requests.Load() != 0 {
				t.Fatalf("noninteractive selection changed options or made requests: err=%v requests=%d", err, requests.Load())
			}
		})
	}
}

func TestChooseCloudStackAutomaticallyUsesOnlyReadyAuthorizedChoice(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/organizations/org/stacks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if _, err := io.WriteString(w, `{"data":[{"id":"ready","name":"Books","status":"READY","state":"ACTIVE"},{"id":"disabled","status":"READY","state":"DISABLED"},{"id":"unsigned","status":"READY"}]}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := api.New(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, cmd := selectionCommand(t, &Settings{})
	stack, err := chooseCloudStack(t.Context(), cmd, client, "org", []string{"ready", "disabled"})
	if err != nil || stack != "ready" || requests.Load() != 1 {
		t.Fatalf("selected stack=%q err=%v requests=%d", stack, err, requests.Load())
	}
}

func TestChooseCloudStackDoesNotFallBackToUnavailableSignedTarget(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, `{"data":[{"id":"only","status":"PROGRESSING"}]}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := api.New(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, cmd := selectionCommand(t, &Settings{})
	if stack, err := chooseCloudStack(t.Context(), cmd, client, "org", []string{"only"}); err == nil || stack != "" {
		t.Fatal("selected an unavailable signed target")
	}
}

func TestChooseCloudOrganizationSingleAndCanceledNeedNoListing(t *testing.T) {
	t.Parallel()
	_, cmd := selectionCommand(t, &Settings{})
	organization, err := chooseCloudOrganization(t.Context(), cmd, nil, []string{"only"})
	if err != nil || organization != "only" {
		t.Fatalf("single organization=%q err=%v", organization, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if organization, err := chooseCloudOrganization(ctx, cmd, nil, []string{"only"}); !errors.Is(err, context.Canceled) || organization != "" {
		t.Fatalf("canceled selection organization=%q err=%v", organization, err)
	}
}

func TestReadCloudChoicesRejectsMalformedData(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, `{"data":[{"id":42}]}`); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := api.New(server.URL, &http.Client{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readCloudChoices(t.Context(), client, "/organizations"); err == nil {
		t.Fatal("accepted malformed resource IDs")
	}
}

type selectionFixture struct {
	server             *httptest.Server
	dir                string
	entry              Entry
	organizations      []cloudChoiceResource
	stacks             map[string][]cloudChoiceResource
	organizationTokens map[string]string
	listOrganizations  atomic.Int32
	listStacks         atomic.Int32
	posts              atomic.Int32
}

func newSelectionFixture(t *testing.T) *selectionFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &selectionFixture{
		dir: t.TempDir(), organizationTokens: make(map[string]string),
		organizations: []cloudChoiceResource{{ID: "first", Name: "Accounts"}, {ID: "second", Name: "Books"}},
		stacks: map[string][]cloudChoiceResource{
			"first":  {{ID: "one", Name: "Sandbox", Status: "READY", State: "ACTIVE"}, {ID: "two", Name: "Production", Status: "READY", State: "ACTIVE"}},
			"second": {{ID: "three", Name: "Books", Status: "READY", State: "ACTIVE"}},
		},
	}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.serve(t, key, w, r)
	}))
	t.Cleanup(f.server.Close)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "key"))
	if err != nil {
		t.Fatal(err)
	}
	issuer := f.server.URL + "/api"
	scopes := []string{"organization:Read", "organization:ListStacks"}
	accesses := []any{
		map[string]any{"id": "first", "scopes": scopes, "stacks": []any{
			map[string]any{"id": "one", "uri": f.server.URL + "/one", "scopes": []string{"stack:Read"}},
			map[string]any{"id": "two", "uri": f.server.URL + "/two", "scopes": []string{"stack:Read"}},
		}},
		map[string]any{"id": "second", "scopes": scopes, "stacks": []any{
			map[string]any{"id": "three", "uri": f.server.URL + "/three", "scopes": []string{"stack:Read"}},
		}},
	}
	id := signSelectionClaims(t, signer, issuer, map[string]any{"org": accesses})
	options := cloud.Options{Issuer: issuer, ClientID: "fctl"}
	root := &cloud.Session{Options: options, IDToken: id, MembershipToken: selectionToken("root-access"), Organizations: make(map[string]*cloud.Session)}
	for _, organization := range []string{"first", "second"} {
		access := signSelectionClaims(t, signer, issuer, map[string]any{"organization_id": organization, "scope": strings.Join(scopes, " ")})
		f.organizationTokens[organization] = access
		childOptions := options
		childOptions.Organization = organization
		root.Organizations[organization] = &cloud.Session{Options: childOptions, IDToken: id, MembershipToken: selectionToken(access)}
	}
	f.entry = NewEntry(Options{AuthMode: "cloud", Issuer: issuer, ClientID: "fctl"})
	f.entry.Session = root
	if err := Save(f.dir, Store{Active: "work", Connections: map[string]Entry{"work": f.entry}}); err != nil {
		t.Fatal(err)
	}
	return f
}

func selectionToken(access string) *oauth2.Token {
	return &oauth2.Token{AccessToken: access, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
}

func signSelectionClaims(t *testing.T, signer jose.Signer, issuer string, values map[string]any) string {
	t.Helper()
	values["iss"], values["aud"], values["sub"], values["exp"] = issuer, "fctl", "user", time.Now().Add(time.Hour).Unix()
	raw, err := jwt.Signed(signer).Claims(values).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (f *selectionFixture) serve(t *testing.T, key *rsa.PrivateKey, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodGet {
		f.posts.Add(1)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	switch r.URL.Path {
	case "/api/.well-known/openid-configuration":
		writeSelectionJSON(t, w, map[string]any{
			"issuer": f.server.URL + "/api", "authorization_endpoint": f.server.URL + "/api/authorize", "device_authorization_endpoint": f.server.URL + "/api/device",
			"token_endpoint": f.server.URL + "/api/token", "jwks_uri": f.server.URL + "/api/jwks", "id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/api/jwks":
		writeSelectionJSON(t, w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "key", Algorithm: "RS256", Use: "sig"}}})
	case "/api/organizations":
		f.listOrganizations.Add(1)
		if r.Header.Get("Authorization") != "Bearer root-access" {
			t.Error("organization listing did not use the root Membership client")
		}
		writeSelectionJSON(t, w, map[string]any{"data": f.organizations})
	case "/api/organizations/first/stacks", "/api/organizations/second/stacks":
		f.listStacks.Add(1)
		organization := strings.Split(r.URL.Path, "/")[3]
		if r.Header.Get("Authorization") != "Bearer "+f.organizationTokens[organization] {
			t.Error("stack listing did not use its scoped Membership grant")
		}
		writeSelectionJSON(t, w, map[string]any{"data": f.stacks[organization]})
	default:
		t.Errorf("unexpected request to %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func writeSelectionJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}

func (f *selectionFixture) settings(t *testing.T) (*Settings, *cobra.Command) {
	t.Helper()
	s := &Settings{NoBrowser: true}
	root, cmd := selectionCommand(t, s)
	if err := root.PersistentFlags().Set("config-dir", f.dir); err != nil {
		t.Fatal(err)
	}
	if err := root.PersistentFlags().Set("no-browser", "true"); err != nil {
		t.Fatal(err)
	}
	return s, cmd
}

func TestCloudSelectionNoninteractivePreservesExistingErrorsAndRootSession(t *testing.T) {
	t.Parallel()
	f := newSelectionFixture(t)
	before, err := os.ReadFile(filepath.Join(f.dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	s, cmd := f.settings(t)
	cmd.Annotations = map[string]string{"fctl.target": "organization"}
	client, err := s.Client(t.Context(), cmd, "cloud")
	if err != nil {
		t.Fatal(err)
	}
	if client.Context()["organization"] != "" || client.Context()["stack"] != "" {
		t.Fatal("selected an ambiguous Cloud management target without input")
	}
	if _, err := s.Client(t.Context(), cmd, "ledger"); err == nil || !strings.HasPrefix(err.Error(), "choose a Cloud stack with --organization") {
		t.Fatalf("old missing-target error changed: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(f.dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || f.listOrganizations.Load() != 0 || f.listStacks.Load() != 0 || f.posts.Load() != 0 {
		t.Fatal("noninteractive resolution mutated the profile or listed resources")
	}
}

type selectionRunner func(context.Context, *cobra.Command, []interactive.Field) ([]string, error)

func (run selectionRunner) Run(ctx context.Context, cmd *cobra.Command, fields []interactive.Field) ([]string, error) {
	return run(ctx, cmd, fields)
}

func selectionContext(t *testing.T, cmd *cobra.Command, run selectionRunner) context.Context {
	t.Helper()
	t.Setenv("CI", "")
	t.Setenv("FCTL_NO_INPUT", "")
	ctx := interactive.WithRunner(t.Context(), run)
	cmd.SetContext(ctx)
	return ctx
}

func (f *selectionFixture) storeBytes(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCloudManagementSelectsOnlyAnnotatedOrganization(t *testing.T) {
	for _, target := range []string{"", "identity", "organization"} {
		t.Run(target, func(t *testing.T) {
			checkCloudManagementTarget(t, target)
		})
	}
}

func checkCloudManagementTarget(t *testing.T, target string) {
	t.Helper()
	f := newSelectionFixture(t)
	before := f.storeBytes(t)
	s, cmd := f.settings(t)
	cmd.Annotations = map[string]string{"fctl.target": target}
	calls := 0
	ctx := selectionContext(t, cmd, func(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
		calls++
		want := []interactive.Option{{Label: "Accounts (first)", Value: "first"}, {Label: "Books (second)", Value: "second"}}
		if target != "organization" || len(fields) != 1 || !reflect.DeepEqual(fields[0].Options, want) {
			t.Fatalf("unexpected picker fields: %+v", fields)
		}
		return []string{"second"}, nil
	})
	client, err := s.Client(ctx, cmd, "cloud")
	if err != nil {
		t.Fatal(err)
	}
	wantOrg, wantCalls := "", 0
	if target == "organization" {
		wantOrg, wantCalls = "second", 1
	}
	if client.Context()["organization"] != wantOrg || client.Context()["stack"] != "" || calls != wantCalls || f.listOrganizations.Load() != int32(wantCalls) || f.listStacks.Load() != 0 || f.posts.Load() != 0 {
		t.Fatal("Cloud command selected an unnecessary target or did not apply the selected organization")
	}
	if f.storeBytes(t) != before {
		t.Fatal("management picker changed the saved defaults or sessions")
	}
}

type stackPickerCase struct {
	name, organization, stack, chosenOrganization, chosenStack string
	answers                                                    []string
	organizationLists, stackLists                              int32
}

func TestCloudStackPickerUsesCoordinatedMembershipAndUniqueChoices(t *testing.T) {
	for _, tc := range []stackPickerCase{
		{name: "organization and stack", chosenOrganization: "first", chosenStack: "two", answers: []string{"first", "two"}, organizationLists: 1, stackLists: 1},
		{name: "unique ready stack", chosenOrganization: "second", chosenStack: "three", answers: []string{"second"}, organizationLists: 1, stackLists: 1},
		{name: "explicit organization", organization: "first", chosenOrganization: "first", chosenStack: "one", answers: []string{"one"}, stackLists: 1},
		{name: "explicit stack identifies organization", stack: "three", chosenOrganization: "second", chosenStack: "three"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkCloudStackPickerCase(t, tc)
		})
	}
}

func checkCloudStackPickerCase(t *testing.T, tc stackPickerCase) {
	t.Helper()
	f := newSelectionFixture(t)
	before := f.storeBytes(t)
	s, cmd := f.settings(t)
	setSelectionFlags(t, cmd, tc.organization, tc.stack)
	calls := 0
	ctx := selectionContext(t, cmd, func(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
		if len(fields) != 1 || calls >= len(tc.answers) {
			t.Fatal("unexpected picker call")
		}
		if fields[0].Title == "Choose a Cloud stack" {
			want := []interactive.Option{{Label: "Production (two) · READY", Value: "two"}, {Label: "Sandbox (one) · READY", Value: "one"}}
			if !reflect.DeepEqual(fields[0].Options, want) {
				t.Fatalf("stack picker options = %+v", fields[0].Options)
			}
		}
		answer := tc.answers[calls]
		calls++
		return []string{answer}, nil
	})
	options, entry, name, dir, err := s.Resolve(cmd)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := s.selectCloudTarget(ctx, cmd, f.server.Client(), options, entry, name, dir)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Organization != tc.chosenOrganization || selected.Stack != tc.chosenStack || calls != len(tc.answers) || f.listOrganizations.Load() != tc.organizationLists || f.listStacks.Load() != tc.stackLists || f.posts.Load() != 0 {
		t.Fatal("selected target or Membership requests differ from expected resolution")
	}
	resolved, _, _, _, err := s.Resolve(cmd)
	if err != nil || resolved != selected || f.storeBytes(t) != before {
		t.Fatal("choice lost its command flags or changed persisted defaults/session")
	}
}

func setSelectionFlags(t *testing.T, cmd *cobra.Command, organization, stack string) {
	t.Helper()
	for name, value := range map[string]string{"organization": organization, "stack": stack} {
		if value != "" {
			if err := cmd.Flags().Set(name, value); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCloudTargetPickerCancellationStopsBeforeServiceAuthorization(t *testing.T) {
	for _, atStack := range []bool{false, true} {
		t.Run(map[bool]string{false: "organization", true: "stack"}[atStack], func(t *testing.T) {
			checkCloudPickerCancellation(t, atStack)
		})
	}
}

func checkCloudPickerCancellation(t *testing.T, atStack bool) {
	t.Helper()
	f := newSelectionFixture(t)
	before := f.storeBytes(t)
	s, cmd := f.settings(t)
	ctx := selectionContext(t, cmd, func(_ context.Context, _ *cobra.Command, fields []interactive.Field) ([]string, error) {
		if atStack && fields[0].Title == "Choose a Cloud organization" {
			return []string{"first"}, nil
		}
		return nil, interactive.ErrCanceled
	})
	if _, err := s.Client(ctx, cmd, "ledger"); !errors.Is(err, interactive.ErrCanceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	wantStacks := int32(0)
	if atStack {
		wantStacks = 1
	}
	if f.listOrganizations.Load() != 1 || f.listStacks.Load() != wantStacks || f.posts.Load() != 0 || f.storeBytes(t) != before || cmd.Flags().Changed("organization") || cmd.Flags().Changed("stack") {
		t.Fatal("canceled picker changed the target/session or authorized a service")
	}
}

func TestResolvedCloudTargetsRespectFlagEnvironmentAndSavedDefaults(t *testing.T) {
	for _, source := range []string{"saved", "environment", "flags"} {
		t.Run(source, func(t *testing.T) {
			checkResolvedCloudTargetSource(t, source)
		})
	}
}

func checkResolvedCloudTargetSource(t *testing.T, source string) {
	t.Helper()
	f := newSelectionFixture(t)
	entry := f.entry
	entry.Options.Organization, entry.Options.Stack = "first", "one"
	if err := Save(f.dir, Store{Active: "work", Connections: map[string]Entry{"work": entry}}); err != nil {
		t.Fatal(err)
	}
	before := f.storeBytes(t)
	s, cmd := f.settings(t)
	wantOrg, wantStack := "first", "one"
	t.Setenv("FCTL_ORGANIZATION", "first")
	t.Setenv("FCTL_STACK", "one")
	if source == "saved" {
		if err := os.Unsetenv("FCTL_ORGANIZATION"); err != nil {
			t.Fatal(err)
		}
		if err := os.Unsetenv("FCTL_STACK"); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Setenv("FCTL_ORGANIZATION", "second")
		t.Setenv("FCTL_STACK", "three")
		wantOrg, wantStack = "second", "three"
	}
	if source == "flags" {
		setSelectionFlags(t, cmd, "first", "two")
		wantOrg, wantStack = "first", "two"
	}
	ctx := selectionContext(t, cmd, func(_ context.Context, _ *cobra.Command, _ []interactive.Field) ([]string, error) {
		t.Fatal("complete resolved target prompted for a choice")
		return nil, interactive.ErrCanceled
	})
	options, original, name, dir, err := s.Resolve(cmd)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := s.selectCloudTarget(ctx, cmd, f.server.Client(), options, original, name, dir)
	if err != nil || selected.Organization != wantOrg || selected.Stack != wantStack || f.listOrganizations.Load() != 0 || f.listStacks.Load() != 0 || f.posts.Load() != 0 || f.storeBytes(t) != before {
		t.Fatal("target precedence changed or a complete target required discovery")
	}
}

func TestCloudPickerHonorsDisabledInputWithInjectedRunner(t *testing.T) {
	for _, disabled := range []string{"no-input", "CI", "FCTL_NO_INPUT"} {
		t.Run(disabled, func(t *testing.T) {
			s := &Settings{}
			_, cmd := selectionCommand(t, s)
			ctx := selectionContext(t, cmd, func(_ context.Context, _ *cobra.Command, _ []interactive.Field) ([]string, error) {
				t.Fatal("disabled input invoked a picker")
				return nil, interactive.ErrCanceled
			})
			if disabled == "no-input" {
				if err := cmd.Flags().Set("no-input", "true"); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Setenv(disabled, "true")
			}
			options := Options{AuthMode: "cloud"}
			entry := Entry{Options: options, Session: &cloud.Session{}}
			selected, err := s.selectCloudTarget(ctx, cmd, nil, options, entry, "work", t.TempDir())
			if err != nil || selected != options {
				t.Fatalf("disabled input changed resolution: %v", err)
			}
		})
	}
}

func TestCloudPickerLeavesLegacySessionBoundaryIntact(t *testing.T) {
	s := &Settings{}
	_, cmd := selectionCommand(t, s)
	ctx := selectionContext(t, cmd, func(_ context.Context, _ *cobra.Command, _ []interactive.Field) ([]string, error) {
		t.Fatal("legacy single-stack session invoked a picker")
		return nil, interactive.ErrCanceled
	})
	options := Options{AuthMode: "cloud"}
	entry := Entry{Options: options, Session: &cloud.Session{Options: cloud.Options{Stack: "legacy"}}}
	selected, err := s.selectCloudTarget(ctx, cmd, nil, options, entry, "work", t.TempDir())
	if err != nil || selected != options {
		t.Fatalf("legacy session changed resolution: %v", err)
	}
}
