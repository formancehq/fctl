package cloud

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
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
)

type membershipFixture struct {
	identity                                *identityFixture
	client                                  *http.Client
	mu                                      sync.Mutex
	forms                                   []url.Values
	grants                                  map[string][]string
	used                                    map[string]bool
	grantScopes, refreshScopes              []string
	wrongOrganization                       bool
	accessAudience, accessSubject           string
	invalidAccess                           bool
	devices, refreshes, rootCalls, orgCalls atomic.Int32
}

func newMembershipFixture(t *testing.T) *membershipFixture {
	t.Helper()
	f := &membershipFixture{identity: newIdentityFixture(t), grants: make(map[string][]string), used: make(map[string]bool)}
	f.identity.accesses[0].Scopes = []string{"organization:Read", "organization:ListStacks", "unrelated"}
	f.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return f.roundTrip(t, r)
	})}
	return f
}

func (f *membershipFixture) roundTrip(t *testing.T, r *http.Request) (*http.Response, error) {
	t.Helper()
	if r.URL.Path == "/membership/device" || r.URL.Path == "/membership/token" {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		if r.Form.Get("refresh_token") == "root-refresh" {
			return f.identity.client.Transport.RoundTrip(r)
		}
		f.mu.Lock()
		f.forms = append(f.forms, r.Form)
		f.mu.Unlock()
		return f.oauthResponse(t, r), nil
	}
	if strings.HasPrefix(r.URL.Path, "/membership/me") || r.URL.Path == "/membership/organizations" || strings.HasPrefix(r.URL.Path, "/membership/organizations/") {
		return f.serviceResponse(t, r), nil
	}
	return f.identity.client.Transport.RoundTrip(r)
}

func (f *membershipFixture) oauthResponse(t *testing.T, r *http.Request) *http.Response {
	t.Helper()
	w := httptest.NewRecorder()
	id, _, ok := r.BasicAuth()
	if !ok || id != "fctl" {
		t.Error("missing organization client authentication")
	}
	if r.URL.Path == "/membership/device" {
		f.devices.Add(1)
		org := r.Form.Get("organization_id")
		f.mu.Lock()
		f.grants[org] = intersectOrganizationScopes(strings.Fields(r.Form.Get("scope")))
		f.mu.Unlock()
		writeJSON(t, w, map[string]any{"device_code": org, "user_code": "ABCD", "verification_uri": f.identity.f.server.URL + "/verify", "expires_in": 600, "interval": 1})
		return w.Result()
	}
	if r.Form.Has("scope") || r.Form.Has("resource") {
		t.Error("organization token request repeated scope or resource")
	}
	org, refresh := r.Form.Get("device_code"), "organization-refresh"
	if r.Form.Get("grant_type") == "refresh_token" {
		f.refreshes.Add(1)
		org, refresh = "org", "rotated-organization-refresh"
		f.mu.Lock()
		used := f.used[r.Form.Get("refresh_token")]
		f.used[r.Form.Get("refresh_token")] = true
		f.mu.Unlock()
		if used {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(t, w, map[string]string{"error": "invalid_grant", "error_description": "rotation-secret"})
			return w.Result()
		}
	}
	writeJSON(t, w, map[string]any{"access_token": f.accessToken(t, org, refresh), "refresh_token": refresh, "token_type": "Bearer", "expires_in": 3600, "id_token": f.identity.signedID(t)})
	return w.Result()
}

func (f *membershipFixture) accessToken(t *testing.T, org, refresh string) string {
	t.Helper()
	f.mu.Lock()
	scopes := slices.Clone(f.grants[org])
	f.mu.Unlock()
	if f.grantScopes != nil {
		scopes = f.grantScopes
	}
	if refresh == "rotated-organization-refresh" && f.refreshScopes != nil {
		scopes = f.refreshScopes
	}
	if f.wrongOrganization {
		org = "other"
	}
	audience, subject := f.accessAudience, f.accessSubject
	if audience == "" {
		audience = "fctl"
	}
	if subject == "" {
		subject = "user"
	}
	raw := signMembershipClaims(t, f.identity, map[string]any{"iss": f.identity.f.options.Issuer, "aud": audience, "sub": subject, "exp": time.Now().Add(time.Hour).Unix(), "organization_id": org, "scope": strings.Join(scopes, " ")})
	if f.invalidAccess {
		raw += "invalid"
	}
	return raw
}

func signMembershipClaims(t *testing.T, f *identityFixture, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	unsigned := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"key"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload)
	hash := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.f.key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func (f *membershipFixture) serviceResponse(t *testing.T, r *http.Request) *http.Response {
	t.Helper()
	w := httptest.NewRecorder()
	if strings.HasPrefix(r.URL.Path, "/membership/organizations/") {
		f.orgCalls.Add(1)
		if r.Header.Get("Authorization") == "Bearer root-secret" {
			t.Error("organization request used root credentials")
		}
	} else {
		f.rootCalls.Add(1)
		if !slices.Contains([]string{"Bearer root-secret", "Bearer refreshed-root-secret"}, r.Header.Get("Authorization")) {
			t.Error("root request used organization credentials")
		}
	}
	writeJSON(t, w, map[string]bool{"ok": true})
	return w.Result()
}

func membershipRequest(t *testing.T, client *http.Client, uri string) error {
	t.Helper()
	resp, err := client.Do(testRequest(t, http.MethodPost, uri, nil))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func TestMembershipRoutingAndConcurrentCache(t *testing.T) {
	f := newMembershipFixture(t)
	root := f.identity.root(t)
	root.Targets = map[string]*Session{"org/stack": f.identity.f.session(t)}
	store := newCoordinatedStore(root)
	client, issuer, info, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if issuer != root.Options.Issuer || !slices.Equal(info.Organizations, []string{"org"}) || !slices.Equal(info.Stacks["org"], []string{"stack"}) {
		t.Fatal("incorrect verified metadata")
	}
	for _, route := range []string{"/me", "/organizations?limit=10&cursor=next", "/me/invitations?limit=10", "/me/invitations/invite/accept", "/me/invitations/invite/reject", "/me/invitations/invite/decline"} {
		if err := membershipRequest(t, client, issuer+route); err != nil {
			t.Fatal(err)
		}
	}
	runParallel(t, func() error {
		other, _, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
		if err != nil {
			return err
		}
		return membershipRequest(t, other, issuer+"/organizations/org/stacks")
	})
	assertOrganizationForms(t, f, root.IDToken)
	saved := roundTripSession(t, store.snapshot())
	if saved.Organizations["org"] == nil || saved.Targets["org/stack"] == nil || f.devices.Load() != 1 || f.orgCalls.Load() != 2 || f.identity.f.exchanges.Load() != 0 {
		t.Fatal("organization cache duplicated, lost targets, or reached stack exchange")
	}
	for _, uri := range []string{"https://other.invalid/organizations/org", issuer + "/organizations/unknown", issuer + "/organizations/org/../other", issuer + "/organizations/org%2Fother", f.identity.f.server.URL + "/me", issuer + "/me/unsupported", issuer + "/me/invitations/invite/unsupported"} {
		if err := membershipRequest(t, client, uri); err == nil {
			t.Fatal("unsafe or unavailable route was accepted")
		}
	}
}

func assertOrganizationForms(t *testing.T, f *membershipFixture, hint string) {
	t.Helper()
	if len(f.forms) != 2 {
		t.Fatal("unexpected organization auth request count")
	}
	if f.forms[0].Get("scope") != "openid offline_access organization:Read organization:ListStacks" || f.forms[0].Get("organization_id") != "org" || f.forms[0].Get("id_token_hint") != hint || f.forms[0].Has("resource") {
		t.Fatal("incorrect organization authorization parameters")
	}
	if f.forms[1].Has("scope") || f.forms[1].Has("resource") {
		t.Fatal("incorrect organization polling parameters")
	}
}

func TestMembershipInitialRefreshUpdatesMetadata(t *testing.T) {
	f := newMembershipFixture(t)
	f.identity.accesses[0].Stacks = nil
	root := f.identity.root(t)
	root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	f.identity.accesses[0].Stacks = []stackAccess{{ID: "new", URI: f.identity.f.server.URL + "/stack", Scopes: []string{"stack:Read"}}}
	store := newCoordinatedStore(root)
	_, _, info, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(info.Stacks["org"], []string{"new"}) || store.snapshot().MembershipToken.RefreshToken != "rotated-root-refresh" || f.devices.Load() != 0 {
		t.Fatal("metadata preceded root refresh or started a device flow")
	}
}

func TestMembershipRejectsIncorrectGrant(t *testing.T) {
	for _, kind := range []string{"organization", "scopes", "signature", "audience", "access signature", "access audience", "access subject"} {
		t.Run(kind, func(t *testing.T) { t.Parallel(); checkIncorrectOrganizationGrant(t, kind) })
	}
}

func checkIncorrectOrganizationGrant(t *testing.T, kind string) {
	t.Helper()
	f := newMembershipFixture(t)
	root := f.identity.root(t)
	store := newCoordinatedStore(root)
	client, issuer, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	switch kind {
	case "organization":
		f.wrongOrganization = true
	case "scopes":
		f.grantScopes = []string{"organization:Read", "organization:Delete"}
	case "signature":
		f.identity.invalidID = true
	case "audience":
		f.identity.f.audience = "other"
	case "access signature":
		f.invalidAccess = true
	case "access audience":
		f.accessAudience = "other"
	case "access subject":
		f.accessSubject = "other"
	}
	if err := membershipRequest(t, client, issuer+"/organizations/org"); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe organization grant result: %v", err)
	}
	if len(store.snapshot().Organizations) != 0 || f.orgCalls.Load() != 0 {
		t.Fatal("invalid grant persisted or reached service")
	}
}

func TestMembershipRefreshRotationAndScopeGuard(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid", true: "elevated"}[invalid], func(t *testing.T) { t.Parallel(); checkOrganizationRefresh(t, invalid) })
	}
}

func checkOrganizationRefresh(t *testing.T, invalid bool) {
	t.Helper()
	f := newMembershipFixture(t)
	root := f.identity.root(t)
	store := newCoordinatedStore(root)
	client, issuer, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if err := membershipRequest(t, client, issuer+"/organizations/org"); err != nil {
		t.Fatal(err)
	}
	f.orgCalls.Store(0)
	store.mu.Lock()
	store.session.Organizations["org"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	store.revision++
	store.mu.Unlock()
	if invalid {
		f.refreshScopes = []string{"organization:Read", "organization:ListStacks", "organization:Delete"}
	}
	if invalid {
		err = membershipRequest(t, client, issuer+"/organizations/org")
	} else {
		runParallel(t, func() error {
			other, _, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
			if err != nil {
				return err
			}
			return membershipRequest(t, other, issuer+"/organizations/org")
		})
	}
	if (err != nil) != invalid {
		t.Fatalf("incorrect refresh result: %v", err)
	}
	saved := store.snapshot().Organizations["org"]
	if saved.MembershipToken.RefreshToken != "rotated-organization-refresh" || f.refreshes.Load() != 1 {
		t.Fatal("refresh rotation lost")
	}
	if invalid {
		assertBlockedOrganization(t, f, store, issuer)
	} else {
		runParallel(t, func() error { return membershipRequest(t, client, issuer+"/organizations/org") })
	}
	if f.refreshes.Load() != 1 || f.identity.f.exchanges.Load() != 0 {
		t.Fatal("refresh retried or exchanged against stack")
	}
}

func assertBlockedOrganization(t *testing.T, f *membershipFixture, store *coordinatedStore, issuer string) {
	t.Helper()
	if store.snapshot().Organizations["org"].IDToken != "" || f.orgCalls.Load() != 0 {
		t.Fatal("invalid refresh retained identity or reached service")
	}
	other, _, _, err := MembershipClient(t.Context(), f.client, roundTripSession(t, store.snapshot()), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if err := membershipRequest(t, other, issuer+"/organizations/org"); err == nil {
		t.Fatal("new client accepted blocked organization identity")
	}
}

func TestMembershipAuthorizationCancellationAndLogout(t *testing.T) {
	for _, logout := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancellation", true: "logout"}[logout], func(t *testing.T) { t.Parallel(); checkOrganizationCancellation(t, logout) })
	}
}

func checkOrganizationCancellation(t *testing.T, logout bool) {
	t.Helper()
	f := newMembershipFixture(t)
	root := f.identity.root(t)
	store := newCoordinatedStore(root)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client, issuer, _, err := MembershipClient(ctx, f.client, copyRoot(root), io.Discard, func(context.Context, string) error {
		if logout {
			store.mu.Lock()
			store.session = nil
			store.revision++
			store.mu.Unlock()
		} else {
			cancel()
		}
		return nil
	}, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	err = membershipRequest(t, client, issuer+"/organizations/org")
	if err == nil || (!logout && !errors.Is(err, context.Canceled)) {
		t.Fatalf("cancellation/logout not respected: %v", err)
	}
	if f.orgCalls.Load() != 0 || (logout && store.snapshot() != nil) {
		t.Fatal("invalidated authorization reached service or restored root")
	}
}

func TestExplicitNewStackRefreshesRootOnce(t *testing.T) {
	f := newIdentityFixture(t)
	f.accesses[0].Stacks = nil
	root := f.root(t)
	f.accesses[0].Stacks = []stackAccess{{ID: "new", URI: f.f.server.URL + "/stack", Scopes: []string{"stack:Read"}}}
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{Stack: "new"}, io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if f.rootRefresh.Load() != 1 || len(f.forms) != 3 || store.snapshot().Targets["org/new"] == nil || store.snapshot().MembershipToken.RefreshToken != "rotated-root-refresh" {
		t.Fatal("new stack did not use a single root refresh followed by scoped authorization")
	}
}

func TestOrganizationCacheRenewsScopeChanges(t *testing.T) {
	for _, upgrade := range []bool{false, true} {
		t.Run(map[bool]string{false: "downgrade", true: "upgrade"}[upgrade], func(t *testing.T) { t.Parallel(); checkOrganizationScopeChange(t, upgrade) })
	}
}

func checkOrganizationScopeChange(t *testing.T, upgrade bool) {
	t.Helper()
	f := newMembershipFixture(t)
	if upgrade {
		f.identity.accesses[0].Scopes = []string{"organization:Read"}
	}
	root := f.identity.root(t)
	store := newCoordinatedStore(root)
	client, issuer, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if err := membershipRequest(t, client, issuer+"/organizations/org"); err != nil {
		t.Fatal(err)
	}
	if upgrade {
		f.identity.accesses[0].Scopes = []string{"organization:Read", "organization:ListStacks"}
	} else {
		f.identity.accesses[0].Scopes = []string{"organization:Read"}
	}
	rootID := f.identity.signedID(t)
	store.mu.Lock()
	store.session.IDToken = rootID
	store.revision++
	store.mu.Unlock()
	if err := membershipRequest(t, client, issuer+"/organizations/org"); err != nil {
		t.Fatal(err)
	}
	if f.devices.Load() != 2 || len(f.forms) != 4 || f.forms[2].Get("id_token_hint") != rootID || f.refreshes.Load() != 0 {
		t.Fatal("scope change did not renew a scoped organization grant")
	}
	want := "openid offline_access " + strings.Join(intersectOrganizationScopes(f.identity.accesses[0].Scopes), " ")
	if f.forms[2].Get("scope") != want {
		t.Fatal("renewed organization scopes differ from signed root")
	}
}

func TestOrganizationScopeFallbackAndDenial(t *testing.T) {
	claims := identityClaims{Organizations: []organizationAccess{{ID: "org"}}}
	scopes, err := organizationScopes(claims, "org")
	if err != nil || !slices.Equal(scopes, knownOrganizationScopes) {
		t.Fatal("legacy root claim fallback omitted known organization scopes")
	}
	claims.Organizations[0].Scopes = []string{}
	if _, err := organizationScopes(claims, "org"); err == nil {
		t.Fatal("explicit empty scope grant was treated as legacy absence")
	}
	claims.Organizations[0].Scopes = []string{"unsupported:scope"}
	if _, err := organizationScopes(claims, "org"); err == nil {
		t.Fatal("unsupported scopes granted organization access")
	}
}

func TestIdentityInfoRejectsDuplicateOrganizations(t *testing.T) {
	options := Options{Issuer: "https://example.invalid", ClientID: "fctl"}
	claims := identityClaims{Organizations: []organizationAccess{{ID: "org"}, {ID: "org"}}}
	if _, err := verifiedIdentityInfo(options, claims); err == nil {
		t.Fatal("duplicate organizations were accepted")
	}
	info, err := verifiedIdentityInfo(options, identityClaims{})
	if err != nil || len(info.Organizations) != 0 || len(info.Stacks) != 0 {
		t.Fatal("zero-access identity metadata was rejected")
	}
}

func TestMembershipSaveFailureStopsRotationRetry(t *testing.T) {
	f := newMembershipFixture(t)
	root := f.identity.root(t)
	store := newCoordinatedStore(root)
	client, issuer, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if err := membershipRequest(t, client, issuer+"/organizations/org"); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.session.Organizations["org"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	store.failSave = true
	store.mu.Unlock()
	for range 2 {
		if err := membershipRequest(t, client, issuer+"/organizations/org"); err == nil {
			t.Fatal("failed CAS did not stop authenticated requests")
		}
	}
	if f.refreshes.Load() != 1 || f.orgCalls.Load() != 1 || store.snapshot().Organizations["org"].MembershipToken.RefreshToken != "organization-refresh" {
		t.Fatal("save failure retried a consumed refresh token or changed stored credentials")
	}
}

func TestMissingExplicitStackDoesNotRefreshTwice(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(map[bool]string{false: "current root", true: "expired root"}[expired], func(t *testing.T) {
			t.Parallel()
			f := newIdentityFixture(t)
			f.accesses[0].Stacks = nil
			root := f.root(t)
			if expired {
				root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
			}
			store := newCoordinatedStore(root)
			_, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{Stack: "missing"}, io.Discard, nil, store.coordinate)
			if err == nil || f.rootRefresh.Load() != 1 || f.childRefresh.Load() != 0 || len(f.forms) != 1 || len(store.snapshot().Targets) != 0 {
				t.Fatal("unavailable explicit stack triggered repeated refresh or authorization")
			}
		})
	}
}

func TestAmbiguousStackDoesNotRefreshRoot(t *testing.T) {
	f := newIdentityFixture(t)
	f.accesses = append(f.accesses, organizationAccess{ID: "other", Stacks: []stackAccess{{ID: "stack", URI: f.f.server.URL + "/stack", Scopes: []string{"stack:Read"}}}})
	root := f.root(t)
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{Stack: "stack"}, io.Discard, nil, store.coordinate)
	if err == nil || !strings.Contains(err.Error(), "--organization") || strings.Contains(err.Error(), "org/stack") || strings.Contains(err.Error(), "other/stack") || f.rootRefresh.Load() != 0 || len(f.forms) != 0 {
		t.Fatal("ambiguous stack refreshed or failed to give concise selection instructions")
	}
}

func TestExistingStackWrongOrganizationDoesNotRefreshRoot(t *testing.T) {
	for _, org := range []string{"missing", "other"} {
		t.Run(org, func(t *testing.T) {
			t.Parallel()
			f := newIdentityFixture(t)
			f.accesses = append(f.accesses, organizationAccess{ID: "other"})
			root := f.root(t)
			store := newCoordinatedStore(root)
			_, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{Organization: org, Stack: "stack"}, io.Discard, nil, store.coordinate)
			if err == nil || f.rootRefresh.Load() != 0 || len(f.forms) != 0 || store.saves != 0 {
				t.Fatal("existing stack with incorrect organization refreshed identity or authorized a target")
			}
		})
	}
}
