package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type applicationFixture struct {
	identity                         *identityFixture
	client                           *http.Client
	mu                               sync.Mutex
	forms                            []url.Values
	backendTokens                    []string
	scopes                           []string
	mutateAccess                     func(map[string]any)
	mutateID                         func(map[string]any)
	invalidSignature, invalidRefresh bool
	omitRefreshID                    bool
	devices, refreshes, backend      atomic.Int32
}

func newApplicationFixture(t *testing.T) *applicationFixture {
	t.Helper()
	f := &applicationFixture{identity: newIdentityFixture(t), scopes: []string{"apps:Read", "apps:Write"}}
	f.identity.accesses[0].Applications = []applicationAccess{{ID: "application-id", Alias: "deploy", Scopes: slices.Clone(f.scopes)}}
	f.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "backend.test" {
			return f.backendResponse(t, r), nil
		}
		if !sameOrigin(f.identity.f.options.Issuer, r.URL.String()) {
			return nil, errors.New("unexpected network target")
		}
		if r.URL.Path != "/membership/device" && r.URL.Path != "/membership/token" {
			return f.identity.client.Transport.RoundTrip(r)
		}
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		if r.Form.Get("refresh_token") == "root-refresh" {
			return f.identity.client.Transport.RoundTrip(r)
		}
		return f.response(t, r), nil
	})}
	return f
}

func (f *applicationFixture) backendResponse(t *testing.T, r *http.Request) *http.Response {
	t.Helper()
	f.backend.Add(1)
	f.mu.Lock()
	f.backendTokens = append(f.backendTokens, r.Header.Get("Authorization"))
	f.mu.Unlock()
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ey") {
		t.Error("backend did not receive signed application access token")
	}
	w := httptest.NewRecorder()
	if r.URL.Path == "/api/redirect" {
		w.Header().Set("Location", "https://other.test/apps")
		w.WriteHeader(http.StatusFound)
		return w.Result()
	}
	writeJSON(t, w, map[string]bool{"ok": true})
	return w.Result()
}

func (f *applicationFixture) response(t *testing.T, r *http.Request) *http.Response {
	t.Helper()
	f.mu.Lock()
	f.forms = append(f.forms, r.Form)
	f.mu.Unlock()
	w := httptest.NewRecorder()
	if r.URL.Path == "/membership/device" {
		f.devices.Add(1)
		writeJSON(t, w, map[string]any{"device_code": "application-code", "user_code": "ABCD", "verification_uri": f.identity.f.server.URL + "/verify", "expires_in": 600, "interval": 1})
		return w.Result()
	}
	if r.Form.Has("scope") {
		t.Error("application token request must omit scope")
	}
	if r.Form.Get("grant_type") != "refresh_token" && r.Form.Get("resource") != applicationResource("deploy", f.scopes) {
		t.Error("application poll must carry scoped resource")
	}
	refresh := "application-refresh"
	if r.Form.Get("grant_type") == "refresh_token" {
		f.refreshes.Add(1)
		refresh = "rotated-application-refresh"
	}
	access := map[string]any{"iss": f.identity.f.options.Issuer, "aud": []string{"https://backend.test/api"}, "sub": "user", "exp": time.Now().Add(time.Hour).Unix(), "organization_id": "org", "scope": strings.Join(f.scopes, " "), "jti": fmt.Sprintf("application-grant-%d", f.devices.Load())}
	id := map[string]any{"iss": f.identity.f.options.Issuer, "aud": "fctl", "sub": "user", "exp": time.Now().Add(time.Hour).Unix(), "resources": []string{applicationResource("deploy", f.scopes)}}
	if f.mutateAccess != nil {
		f.mutateAccess(access)
	}
	if f.mutateID != nil {
		f.mutateID(id)
	}
	if f.invalidRefresh && refresh == "rotated-application-refresh" {
		access["organization_id"] = "other"
	}
	raw := signMembershipClaims(t, f.identity, access)
	if f.invalidSignature {
		raw += "invalid"
	}
	tokens := map[string]any{"access_token": raw, "id_token": signMembershipClaims(t, f.identity, id), "token_type": "Bearer", "expires_in": 3600, "refresh_token": refresh}
	if f.omitRefreshID && refresh == "rotated-application-refresh" {
		delete(tokens, "id_token")
	}
	writeJSON(t, w, tokens)
	return w.Result()
}

func (f *applicationFixture) login(t *testing.T, root *Session, coordinator Coordinator) (*http.Client, string, error) {
	t.Helper()
	return ApplicationClient(t.Context(), f.client, root, "org", "deploy", io.Discard, nil, coordinator)
}

func TestApplicationGrantAndCache(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	root.Applications = map[string]*Session{"other/other": {Application: "cached-other"}}
	root.Organizations = map[string]*Session{"other": {Application: "sentinel"}}
	root.Targets = map[string]*Session{"other/stack": {StackURL: "sentinel"}}
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "https://backend.test/api" {
		t.Fatalf("unexpected endpoint %s", endpoint)
	}
	if err := membershipRequest(t, client, endpoint+"/apps"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.login(t, root, nil); err != nil {
		t.Fatal(err)
	}
	if f.devices.Load() != 1 || f.identity.f.exchanges.Load() != 0 {
		t.Fatal("application cache missed or stack token exchange occurred")
	}
	child := root.Applications["org/application-id/deploy"]
	if child == nil || child.Application != "deploy" || child.Options.Stack != "" || child.Options.Organization != "org" || child.StackToken != nil {
		t.Fatal("invalid application cache")
	}
	if root.Applications["other/other"].Application != "cached-other" || root.Organizations["other"].Application != "sentinel" || root.Targets["other/stack"].StackURL != "sentinel" {
		t.Fatal("unrelated child cache lost")
	}
	form := f.forms[0]
	if form.Get("scope") != "openid offline_access" || form.Get("organization_id") != "org" || form.Get("id_token_hint") != root.IDToken || form.Get("resource") != "app://deploy|apps:Read apps:Write" {
		t.Fatal("invalid application authorization parameters")
	}
}

func TestApplicationRejectsInvalidGrants(t *testing.T) {
	tests := []struct {
		name   string
		change func(*applicationFixture)
	}{
		{"signature", func(f *applicationFixture) { f.invalidSignature = true }},
		{"issuer", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["iss"] = "https://other.test" }
		}},
		{"expiry", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }
		}},
		{"no backend audience", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["aud"] = []string{"fctl"} }
		}},
		{"missing audience", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { delete(c, "aud") }
		}},
		{"foreign client audience", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["aud"] = []string{"other-client", "https://backend.test/api"} }
		}},
		{"duplicate client audience", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["aud"] = []string{"fctl", "fctl", "https://backend.test/api"} }
		}},
		{"duplicate backend audience", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["aud"] = []string{"https://backend.test/api", "https://backend.test/api"} }
		}},
		{"ambiguous endpoint", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["aud"] = []string{"fctl", "https://backend.test/api", "https://other.test"} }
		}},
		{"http endpoint", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["aud"] = []string{"fctl", "http://localhost/api"} }
		}},
		{"organization", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["organization_id"] = "other" }
		}},
		{"subject", func(f *applicationFixture) { f.mutateAccess = func(c map[string]any) { c["sub"] = "other" } }},
		{"scope escalation", func(f *applicationFixture) {
			f.mutateAccess = func(c map[string]any) { c["scope"] = "apps:Read apps:Write system:Admin" }
		}},
		{"resource alias", func(f *applicationFixture) {
			f.mutateID = func(c map[string]any) { c["resources"] = []string{"app://other|apps:Read apps:Write"} }
		}},
		{"missing proof", func(f *applicationFixture) { f.mutateID = func(c map[string]any) { delete(c, "resources") } }},
		{"id audience", func(f *applicationFixture) { f.mutateID = func(c map[string]any) { c["aud"] = "other" } }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newApplicationFixture(t)
			tt.change(f)
			root := f.identity.root(t)
			if _, _, err := f.login(t, root, nil); err == nil {
				t.Fatal("invalid grant accepted")
			}
			if f.backend.Load() != 0 || len(root.Applications) != 0 {
				t.Fatal("invalid grant used or saved")
			}
		})
	}
}

func TestApplicationRequestBoundsAndRevocation(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, uri := range []string{"http://backend.test/api/apps", "https://other.test/api/apps", "https://backend.test/outside", "https://backend.test/api2/apps", "https://backend.test/api/../outside", "https://backend.test/api/%2e%2e/outside", "https://user@backend.test/api/apps"} {
		if err := membershipRequest(t, client, uri); err == nil {
			t.Fatalf("unsafe URL accepted: %s", uri)
		}
	}
	hostRequest := testRequest(t, http.MethodGet, endpoint+"/apps", nil)
	hostRequest.Host = "other.test"
	hostResponse, hostErr := client.Do(hostRequest)
	if hostResponse != nil {
		if closeErr := hostResponse.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if hostErr == nil {
		t.Fatal("request Host override accepted")
	}
	if f.backend.Load() != 0 {
		t.Fatal("unsafe request reached backend")
	}
	f.identity.accesses[0].Applications = nil
	root.IDToken = f.identity.signedID(t)
	if err := membershipRequest(t, client, endpoint+"/apps"); err == nil {
		t.Fatal("revoked root application used")
	}
	if f.backend.Load() != 0 {
		t.Fatal("revoked request reached backend")
	}
}

func TestApplicationReadOnly(t *testing.T) {
	f := newApplicationFixture(t)
	f.scopes = []string{"apps:Read"}
	f.identity.accesses[0].Applications[0].Scopes = slices.Clone(f.scopes)
	client, endpoint, err := f.login(t, f.identity.root(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(testRequest(t, http.MethodGet, endpoint+"/apps", nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if err := membershipRequest(t, client, endpoint+"/apps"); err == nil {
		t.Fatal("read-only identity authorized write")
	}
	if f.backend.Load() != 1 {
		t.Fatal("unexpected backend calls")
	}
}

func TestApplicationInvalidRefreshPersistsRotation(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	root.Applications["org/application-id/deploy"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	f.invalidRefresh = true
	if err := membershipRequest(t, client, endpoint+"/apps"); err == nil {
		t.Fatal("invalid refreshed grant accepted")
	}
	child := root.Applications["org/application-id/deploy"]
	if child.IDToken != "" || child.MembershipToken.RefreshToken != "rotated-application-refresh" {
		t.Fatal("rotation or invalid-proof marker lost")
	}
	if err := membershipRequest(t, client, endpoint+"/apps"); err == nil {
		t.Fatal("invalid proof reused")
	}
	if _, _, err := f.login(t, root, nil); err == nil {
		t.Fatal("invalid persisted proof reused by new client")
	}
	if f.refreshes.Load() != 1 || f.backend.Load() != 0 {
		t.Fatal("rotation retried or invalid grant forwarded")
	}
}

func TestApplicationCASFailureStopsClient(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	var saves int
	coordinator := func(ctx context.Context, work func(*Session, func(*Session) error) (*oauth2.Token, error)) (*oauth2.Token, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return work(copyRoot(root), func(next *Session) error {
			saves++
			if saves > 1 {
				return errors.New("save-secret")
			}
			*root = *copyRoot(next)
			return nil
		})
	}
	client, endpoint, err := f.login(t, root, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	root.Applications["org/application-id/deploy"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	for range 2 {
		err := membershipRequest(t, client, endpoint+"/apps")
		if err == nil || strings.Contains(err.Error(), "save-secret") {
			t.Fatal("CAS failure accepted or leaked")
		}
	}
	if f.refreshes.Load() != 1 || saves != 2 || f.backend.Load() != 0 {
		t.Fatal("CAS failure retried OAuth or forwarded a request")
	}
}

func TestApplicationCoordinatedRefresh(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	stored := copyRoot(root)
	var mu sync.Mutex
	coordinator := func(ctx context.Context, work func(*Session, func(*Session) error) (*oauth2.Token, error)) (*oauth2.Token, error) {
		mu.Lock()
		defer mu.Unlock()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return work(copyRoot(stored), func(next *Session) error { stored = copyRoot(next); return nil })
	}
	first, endpoint, err := f.login(t, copyRoot(root), coordinator)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := f.login(t, copyRoot(root), coordinator)
	if err != nil {
		t.Fatal(err)
	}
	stored.Applications["org/application-id/deploy"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	failures := make(chan error, 8)
	var wg sync.WaitGroup
	for i := range 8 {
		client := first
		if i%2 != 0 {
			client = second
		}
		wg.Go(func() { failures <- membershipRequest(t, client, endpoint+"/apps") })
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.refreshes.Load() != 1 || f.devices.Load() != 1 || f.backend.Load() != 8 {
		t.Fatal("coordinator did not serialize child refresh across clients")
	}
}

func TestApplicationRootRefresh(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	if err := membershipRequest(t, client, endpoint+"/apps"); err != nil {
		t.Fatal(err)
	}
	if f.identity.rootRefresh.Load() != 1 || f.devices.Load() != 1 || root.Applications["org/application-id/deploy"] == nil {
		t.Fatal("root refresh lost application cache")
	}
}

func TestApplicationCanceledLifetimeAndRedirect(t *testing.T) {
	f := newApplicationFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client, endpoint, err := ApplicationClient(ctx, f.client, f.identity.root(t), "org", "deploy", io.Discard, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(testRequest(t, http.MethodGet, endpoint+"/redirect", nil))
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if err == nil || f.backend.Load() != 1 {
		t.Fatal("application redirect was followed")
	}
	cancel()
	if err := membershipRequest(t, client, endpoint+"/apps"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled lifetime allowed request: %v", err)
	}
	if f.backend.Load() != 1 {
		t.Fatal("canceled request reached backend")
	}
}

func TestApplicationTamperedCache(t *testing.T) {
	for _, field := range []string{"endpoint", "alias", "organization", "stack", "proof", "access"} {
		t.Run(field, func(t *testing.T) {
			f := newApplicationFixture(t)
			root := f.identity.root(t)
			_, _, err := f.login(t, root, nil)
			if err != nil {
				t.Fatal(err)
			}
			child := root.Applications["org/application-id/deploy"]
			switch field {
			case "endpoint":
				child.StackURL = "https://other.test"
			case "alias":
				child.Application = "other"
			case "organization":
				child.Options.Organization = "other"
			case "stack":
				child.Options.Stack = "stack"
			case "proof":
				child.IDToken = f.identity.signedID(t)
			case "access":
				child.MembershipToken.AccessToken = "root-secret"
			}
			if _, _, err := f.login(t, root, nil); err == nil {
				t.Fatal("tampered application cache accepted")
			}
			if f.devices.Load() != 1 || f.backend.Load() != 0 {
				t.Fatal("tampered cache triggered OAuth or backend")
			}
		})
	}
}

func TestApplicationScopesAndEndpointValidation(t *testing.T) {
	for _, claims := range []identityClaims{
		{},
		{Organizations: []organizationAccess{{ID: "org", Applications: []applicationAccess{{ID: "id", Alias: "deploy", Scopes: []string{"other"}}}}}},
		{Organizations: []organizationAccess{{ID: "org", Applications: []applicationAccess{{ID: "id", Alias: "deploy", Scopes: []string{"apps:Read"}}, {ID: "id2", Alias: "deploy", Scopes: []string{"apps:Read"}}}}}},
	} {
		if _, err := applicationGrant(claims, "org", "deploy"); err == nil {
			t.Fatal("invalid root application access accepted")
		}
	}
	for _, endpoint := range []string{"http://localhost/api", "https://user@backend.test/api", "https://backend.test/api?token=x", "https://backend.test/api#fragment", "https://backend.test/api/../outside", "https://backend.test/api%2foutside"} {
		if _, err := applicationEndpoint([]string{"fctl", endpoint}, "fctl"); err == nil {
			t.Fatalf("invalid audience endpoint accepted: %s", endpoint)
		}
	}
	for _, endpoint := range []string{"https://backend.test", "https://backend.test/", "https://backend.test/api/"} {
		if _, err := applicationEndpoint([]string{"fctl", endpoint}, "fctl"); err != nil {
			t.Fatalf("valid audience endpoint rejected: %v", err)
		}
	}
	if !matchesApplicationResource("app://deploy|apps:Write apps:Read", "deploy", []string{"apps:Read", "apps:Write"}) {
		t.Fatal("scope order changed grant semantics")
	}
	if matchesApplicationResource("app://deploy|apps:Read apps:Read", "deploy", []string{"apps:Read"}) {
		t.Fatal("duplicate scope accepted")
	}
}

func TestApplicationRefreshKeepsStoredProof(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	child := root.Applications["org/application-id/deploy"]
	proof := signMembershipClaims(t, f.identity, map[string]any{"iss": root.Options.Issuer, "aud": "fctl", "sub": "user", "exp": time.Now().Add(-time.Hour).Unix(), "resources": []string{"app://deploy|apps:Read apps:Write"}})
	child.IDToken = proof
	child.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	f.omitRefreshID = true
	if err := membershipRequest(t, client, endpoint+"/apps"); err != nil {
		t.Fatal(err)
	}
	if root.Applications["org/application-id/deploy"].IDToken != proof || f.refreshes.Load() != 1 || f.backend.Load() != 1 {
		t.Fatal("refresh lost the verified stored authorization proof")
	}
}

func TestApplicationEndpointChangeDuringRefresh(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	root.Applications["org/application-id/deploy"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	f.mutateAccess = func(claims map[string]any) { claims["aud"] = []string{"fctl", "https://other.test/api"} }
	if err := membershipRequest(t, client, endpoint+"/apps"); err == nil {
		t.Fatal("refresh changed the client backend")
	}
	child := root.Applications["org/application-id/deploy"]
	if child.IDToken != "" || child.MembershipToken.RefreshToken != "rotated-application-refresh" || f.backend.Load() != 0 {
		t.Fatal("endpoint change did not persist rotation and invalidate proof")
	}
}

func TestApplicationAliasReuseRequiresNewGrant(t *testing.T) {
	for _, endpoint := range []string{"https://backend.test/api", "https://backend.test/replacement"} {
		t.Run(endpoint, func(t *testing.T) { checkApplicationAliasReuse(t, endpoint) })
	}
}
func checkApplicationAliasReuse(t *testing.T, endpoint string) {
	t.Helper()
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	first, originalEndpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldToken := root.Applications["org/application-id/deploy"].MembershipToken.AccessToken
	f.identity.accesses[0].Applications[0].ID = "replacement-id"
	root.IDToken = f.identity.signedID(t)
	f.mutateAccess = func(claims map[string]any) { claims["aud"] = []string{"fctl", endpoint} }
	if endpoint == originalEndpoint {
		applicationGet(t, first, originalEndpoint+"/apps")
		applicationAssertNoToken(t, f, oldToken)
	}
	second, newEndpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if newEndpoint != endpoint || f.devices.Load() != 2 {
		t.Fatal("alias reassignment reused its previous application grant")
	}
	if root.Applications["org/replacement-id/deploy"] == nil {
		t.Fatal("replacement grant not bound to its application ID")
	}
	applicationGet(t, second, newEndpoint+"/apps")
	applicationAssertNoToken(t, f, oldToken)
	if endpoint != originalEndpoint {
		if err := membershipRequest(t, first, originalEndpoint+"/apps"); err == nil {
			t.Fatal("existing client changed backend")
		}
		if f.backend.Load() != 1 {
			t.Fatal("old client forwarded a request after backend replacement")
		}
	}
}

func TestApplicationLegacyAliasCacheIsNotMigrated(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	if _, _, err := f.login(t, root, nil); err != nil {
		t.Fatal(err)
	}
	cached := root.Applications["org/application-id/deploy"]
	root.Applications["org/deploy"] = cached
	delete(root.Applications, "org/application-id/deploy")
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	applicationGet(t, client, endpoint+"/apps")
	if f.devices.Load() != 2 {
		t.Fatal("unbound legacy cache was migrated without fresh authorization")
	}
	applicationAssertNoToken(t, f, cached.MembershipToken.AccessToken)
}

func TestApplicationScopeChangesReauthorize(t *testing.T) {
	for _, test := range []struct {
		name          string
		before, after []string
	}{
		{"downgrade", []string{"apps:Read", "apps:Write"}, []string{"apps:Read"}},
		{"upgrade", []string{"apps:Read"}, []string{"apps:Read", "apps:Write"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			checkApplicationScopeChange(t, test.before, test.after)
		})
	}
}

func checkApplicationScopeChange(t *testing.T, before, after []string) {
	t.Helper()
	f := newApplicationFixture(t)
	f.scopes = slices.Clone(before)
	f.identity.accesses[0].Applications[0].Scopes = slices.Clone(before)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	previous := root.Applications["org/application-id/deploy"].MembershipToken.AccessToken
	f.scopes = slices.Clone(after)
	f.identity.accesses[0].Applications[0].Scopes = slices.Clone(after)
	root.IDToken = f.identity.signedID(t)
	applicationGet(t, client, endpoint+"/apps")
	applicationAssertNoToken(t, f, previous)
	if f.devices.Load() != 2 || f.refreshes.Load() != 0 {
		t.Fatal("scope change must use fresh authorization, not the old refresh grant")
	}
	writeErr := membershipRequest(t, client, endpoint+"/apps")
	if (writeErr == nil) != slices.Contains(after, "apps:Write") {
		t.Fatalf("write permission after scope change: %v", writeErr)
	}
	applicationAssertNoToken(t, f, previous)
	if _, _, err := f.login(t, root, nil); err != nil {
		t.Fatal(err)
	}
	if f.devices.Load() != 2 {
		t.Fatal("replacement grant was not cached")
	}
}

func TestApplicationScopeChangeKeepsEndpointImmutable(t *testing.T) {
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := root.Applications["org/application-id/deploy"]
	f.scopes = []string{"apps:Read"}
	f.identity.accesses[0].Applications[0].Scopes = slices.Clone(f.scopes)
	root.IDToken = f.identity.signedID(t)
	f.mutateAccess = func(claims map[string]any) { claims["aud"] = []string{"fctl", "https://backend.test/other"} }
	requestErr := applicationGetError(t, client, endpoint+"/apps")
	if requestErr == nil || !strings.Contains(requestErr.Error(), "endpoint changed") {
		t.Fatalf("endpoint mutation accepted: %v", requestErr)
	}
	if f.devices.Load() != 2 || f.backend.Load() != 0 || root.Applications["org/application-id/deploy"].MembershipToken.AccessToken != before.MembershipToken.AccessToken {
		t.Fatal("failed replacement was used or saved")
	}
}

func TestApplicationScopeChangeDoesNotHideInvalidCache(t *testing.T) {
	for _, field := range []string{"proof signature", "access signature", "proof resource", "proof scopes", "invalidated proof", "JWKS"} {
		t.Run(field, func(t *testing.T) {
			checkInvalidApplicationCache(t, field)
		})
	}
}

func checkInvalidApplicationCache(t *testing.T, field string) {
	t.Helper()
	f := newApplicationFixture(t)
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	child := root.Applications["org/application-id/deploy"]
	f.scopes = []string{"apps:Read"}
	f.identity.accesses[0].Applications[0].Scopes = slices.Clone(f.scopes)
	root.IDToken = f.identity.signedID(t)
	if corruptApplicationCache(t, f, root, child, field) {
		client = nil
	}

	if client == nil {
		if _, _, err := f.login(t, root, nil); err == nil {
			t.Fatal("JWKS failure accepted")
		}
	} else if err := applicationGetError(t, client, endpoint+"/apps"); err == nil {
		t.Fatal("invalid cached proof accepted as a scope change")
	}
	if f.devices.Load() != 1 || f.refreshes.Load() != 0 || f.backend.Load() != 0 {
		t.Fatal("invalid cache triggered fallback authorization or sent a token")
	}
}

func corruptApplicationCache(t *testing.T, f *applicationFixture, root, child *Session, field string) bool {
	t.Helper()
	switch field {
	case "proof signature":
		child.IDToken += "invalid"
	case "access signature":
		child.MembershipToken.AccessToken += "invalid"
	case "proof resource":
		child.IDToken = f.identity.signedID(t)
	case "proof scopes":
		child.IDToken = signMembershipClaims(t, f.identity, map[string]any{"iss": root.Options.Issuer, "aud": "fctl", "sub": "user", "exp": time.Now().Add(time.Hour).Unix(), "resources": []string{"app://deploy|apps:Read"}})
	case "invalidated proof":
		child.IDToken = ""
	case "JWKS":
		base := f.client.Transport
		f.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/membership/keys" {
				return nil, errors.New("synthetic-jwks-failure")
			}
			return base.RoundTrip(req)
		})
		// The existing verifier has a warm key cache. A fresh client has to load JWKS.
		return true
	}
	return false
}

func applicationGet(t *testing.T, client *http.Client, uri string) {
	t.Helper()
	if err := applicationGetError(t, client, uri); err != nil {
		t.Fatal(err)
	}
}
func applicationGetError(t *testing.T, client *http.Client, uri string) error {
	t.Helper()
	response, err := client.Do(testRequest(t, http.MethodGet, uri, nil))
	if response != nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	return err
}

func applicationAssertNoToken(t *testing.T, f *applicationFixture, old string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, token := range f.backendTokens {
		if token == "Bearer "+old {
			t.Fatal("previous application's access token reached the backend")
		}
	}
}

// Membership's application resource resolves to the backend audience, while its
// separate ID token is issued to fctl. Do not verify an access token as an ID token.
func TestApplicationBackendOnlyAccessAudience(t *testing.T) {
	f := newApplicationFixture(t)
	f.mutateAccess = func(claims map[string]any) { claims["aud"] = []string{"https://backend.test/api"} }
	root := f.identity.root(t)
	client, endpoint, err := f.login(t, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	applicationGet(t, client, endpoint+"/apps")
	if root.Applications["org/application-id/deploy"] == nil {
		t.Fatal("backend-only audience grant was not cached by organization/ID/alias")
	}
}

func TestApplicationLegacyClientAndBackendAccessAudience(t *testing.T) {
	f := newApplicationFixture(t)
	f.mutateAccess = func(claims map[string]any) { claims["aud"] = []string{"fctl", "https://backend.test/api"} }
	client, endpoint, err := f.login(t, f.identity.root(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	applicationGet(t, client, endpoint+"/apps")
}
