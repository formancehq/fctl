package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type userInfoFixture struct {
	identity           *identityFixture
	client             *http.Client
	fallback           http.RoundTripper
	access, endpoint   string
	wrongSubject, wait bool
	status             int
	calls              atomic.Int32
}

func newUserInfoFixture(t *testing.T) *userInfoFixture {
	t.Helper()
	f := &userInfoFixture{identity: newIdentityFixture(t)}
	f.identity.omitRefreshID = true
	f.endpoint = f.identity.f.server.URL + "/membership/userinfo"
	f.access = signMembershipClaims(t, f.identity, map[string]any{"iss": f.identity.f.options.Issuer, "aud": "fctl", "sub": "user", "exp": time.Now().Add(time.Hour).Unix()})
	f.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { return f.roundTrip(t, r) })}
	return f
}

func (f *userInfoFixture) roundTrip(t *testing.T, r *http.Request) (*http.Response, error) {
	t.Helper()
	w := httptest.NewRecorder()
	switch r.URL.Path {
	case "/membership/.well-known/openid-configuration":
		writeJSON(t, w, map[string]any{"issuer": f.identity.f.options.Issuer, "token_endpoint": f.identity.f.server.URL + "/membership/token", "device_authorization_endpoint": f.identity.f.server.URL + "/membership/device", "jwks_uri": f.identity.f.server.URL + "/membership/keys", "userinfo_endpoint": f.endpoint, "id_token_signing_alg_values_supported": []string{"RS256"}})
		return w.Result(), nil
	case "/membership/userinfo":
		f.calls.Add(1)
		if f.status != 0 {
			w.WriteHeader(f.status)
			writeJSON(t, w, map[string]string{"error": "userinfo-secret"})
			return w.Result(), nil
		}
		if r.Header.Get("Authorization") != "Bearer "+f.access {
			t.Error("userinfo did not use current root credentials")
		}
		if f.wait {
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		subject := "user"
		if f.wrongSubject {
			subject = "other"
		}
		writeJSON(t, w, map[string]any{"sub": subject, "org": f.identity.accesses})
		return w.Result(), nil
	case "/membership/token":
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		if r.Form.Get("refresh_token") == "root-refresh" {
			return f.rootRefresh(t, r)
		}
	}
	if f.fallback != nil {
		return f.fallback.RoundTrip(r)
	}
	return f.identity.client.Transport.RoundTrip(r)
}

func (f *userInfoFixture) rootRefresh(t *testing.T, r *http.Request) (*http.Response, error) {
	t.Helper()
	resp, err := f.identity.client.Transport.RoundTrip(r)
	if err != nil || resp.StatusCode != http.StatusOK {
		return resp, err
	}
	var body map[string]any
	err = json.NewDecoder(resp.Body).Decode(&body)
	closeErr := resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	body["access_token"] = f.access
	w := httptest.NewRecorder()
	writeJSON(t, w, body)
	return w.Result(), nil
}

func TestRootUserInfoRenewsExpiredIdentityWithoutNewLogin(t *testing.T) {
	f := newUserInfoFixture(t)
	f.identity.idExpiry = time.Now().Add(-time.Hour)
	f.identity.accesses[0].Stacks = nil
	root := f.identity.root(t)
	root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	f.identity.idExpiry = time.Time{}
	f.identity.accesses[0].Stacks = []stackAccess{{ID: "new", URI: f.identity.f.server.URL + "/stack", Scopes: []string{"stack:Read"}}}
	store := newCoordinatedStore(root)
	_, _, info, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(info.Stacks["org"], []string{"new"}) || f.identity.rootRefresh.Load() != 1 || f.calls.Load() != 1 {
		t.Fatal("expired identity did not use current authenticated userinfo after one refresh")
	}
	restored := roundTripSession(t, store.snapshot())
	if restored.IDToken != root.IDToken || restored.MembershipToken.RefreshToken != "rotated-root-refresh" {
		t.Fatal("userinfo replaced signed proof or lost rotation")
	}
	if _, _, _, err := MembershipClient(t.Context(), f.client, restored, io.Discard, nil, store.coordinate); err != nil {
		t.Fatal(err)
	}
	if f.identity.rootRefresh.Load() != 1 || f.calls.Load() != 2 || len(f.identity.forms) != 1 {
		t.Fatal("reloaded client reused unsigned claims or repeated root refresh/device login")
	}
}

func TestNewStackRefreshWithoutIDTokenUsesCurrentUserInfo(t *testing.T) {
	f := newUserInfoFixture(t)
	f.identity.accesses[0].Stacks = nil
	root := f.identity.root(t)
	f.identity.accesses[0].Stacks = []stackAccess{{ID: "new", URI: f.identity.f.server.URL + "/stack", Scopes: []string{"stack:Read"}}}
	store := newCoordinatedStore(root)
	client, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{Organization: "org", Stack: "new"}, io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	if token, err := StackAccessToken(t.Context(), client); err != nil || token != "stack-secret" {
		t.Fatal("second coordinated prepare could not use cached new-stack authorization")
	}
	if store.snapshot().Targets["org/new"] == nil || f.identity.rootRefresh.Load() != 1 || f.calls.Load() < 3 || f.identity.f.exchanges.Load() != 1 || len(f.identity.forms) != 3 {
		t.Fatal("new stack used stale login claims or required a new root login")
	}
}

func organizationUserInfoFixture(t *testing.T) (*userInfoFixture, *membershipFixture) {
	t.Helper()
	f := newUserInfoFixture(t)
	membership := &membershipFixture{identity: f.identity, grants: make(map[string][]string), used: make(map[string]bool)}
	f.fallback = roundTripFunc(func(r *http.Request) (*http.Response, error) { return membership.roundTrip(t, r) })
	return f, membership
}

func TestNewOrganizationUsesFreshUserInfoAcrossRequests(t *testing.T) {
	t.Run("valid root", func(t *testing.T) { checkNewOrganizationRequests(t, false) })
	t.Run("root already refreshed", func(t *testing.T) { checkNewOrganizationRequests(t, true) })
}

func checkNewOrganizationRequests(t *testing.T, expiredRoot bool) {
	t.Helper()
	f, membership := organizationUserInfoFixture(t)
	root := f.identity.root(t)
	if expiredRoot {
		root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	}
	store := newCoordinatedStore(root)
	client, issuer, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	f.identity.accesses = append(f.identity.accesses, organizationAccess{ID: "new", Scopes: []string{"organization:Read"}})
	for range 2 {
		if err := membershipRequest(t, client, issuer+"/organizations/new?expand=true"); err != nil {
			t.Fatal(err)
		}
	}
	saved := roundTripSession(t, store.snapshot())
	if saved.Organizations["new"] == nil || saved.IDToken != root.IDToken || saved.MembershipToken.RefreshToken != "rotated-root-refresh" {
		t.Fatal("new organization authorization lost historical proof, scoped cache or root rotation")
	}
	if f.identity.rootRefresh.Load() != 1 || f.calls.Load() != 2 || membership.devices.Load() != 1 || membership.orgCalls.Load() != 2 || f.identity.f.exchanges.Load() != 0 {
		t.Fatal("new organization repeated root rotation, consent or used stale claims")
	}
	assertNewOrganizationForms(t, membership.forms)
}

func assertNewOrganizationForms(t *testing.T, forms []url.Values) {
	t.Helper()
	if len(forms) != 2 || forms[0].Get("organization_id") != "new" || forms[0].Get("scope") != "openid offline_access organization:Read" || forms[0].Has("resource") || forms[1].Has("scope") {
		t.Fatal("new organization did not use an organization-scoped device grant")
	}
}

func TestMissingOrganizationRefreshIsBoundedAndFailClosed(t *testing.T) {
	f, membership := organizationUserInfoFixture(t)
	root := f.identity.root(t)
	store := newCoordinatedStore(root)
	client, issuer, _, err := MembershipClient(t.Context(), f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := membershipRequest(t, client, issuer+"/organizations/missing"); err == nil {
			t.Fatal("unavailable organization was authorized")
		}
	}
	if f.identity.rootRefresh.Load() != 1 || f.calls.Load() != 2 || membership.devices.Load() != 0 || membership.orgCalls.Load() != 0 || len(store.snapshot().Organizations) != 0 {
		t.Fatal("missing organization repeated rotation or reached authorization/service")
	}
}

func TestRootUserInfoRejectsUnverifiedOrMismatchedIdentity(t *testing.T) {
	for _, kind := range []string{"signature", "audience", "access subject", "userinfo subject", "expired access", "cross origin", "cancellation"} {
		t.Run(kind, func(t *testing.T) { t.Parallel(); checkRootUserInfoFailure(t, kind) })
	}
}

func TestForcedRootRefreshFailureRemainsBlockedAfterReload(t *testing.T) {
	for _, kind := range []string{"signature", "userinfo subject"} {
		t.Run(kind, func(t *testing.T) { checkForcedRootRefreshFailure(t, kind) })
	}
}

func checkForcedRootRefreshFailure(t *testing.T, kind string) {
	t.Helper()
	f := newUserInfoFixture(t)
	root := f.identity.root(t)
	if kind == "signature" {
		f.access += "invalid"
	} else {
		f.wrongSubject = true
	}
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{Stack: "new"}, io.Discard, nil, store.coordinate)
	if err == nil {
		t.Fatal("rejected fresh root proof was accepted")
	}
	restored := roundTripSession(t, store.snapshot())
	if restored.IDToken != "" || restored.MembershipToken.RefreshToken != "rotated-root-refresh" {
		t.Fatal("failed proof retained usable historical claims or lost rotation")
	}
	_, _, _, err = MembershipClient(t.Context(), f.client, restored, io.Discard, nil, store.coordinate)
	if err == nil || f.identity.rootRefresh.Load() != 1 || len(store.snapshot().Targets) != 0 {
		t.Fatal("reloaded client accepted rejected proof or consumed another refresh")
	}
}

func TestForcedRootRefreshTransientFailureRequiresUserInfoAfterReload(t *testing.T) {
	f := newUserInfoFixture(t)
	root := f.identity.root(t)
	f.status = http.StatusServiceUnavailable
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{Stack: "new"}, io.Discard, nil, store.coordinate)
	if err == nil || strings.Contains(err.Error(), "userinfo-secret") {
		t.Fatal("transient userinfo failure was accepted or disclosed its body")
	}
	restored := roundTripSession(t, store.snapshot())
	if !restored.RequireUserInfo || restored.IDToken != root.IDToken || restored.MembershipToken.RefreshToken != "rotated-root-refresh" {
		t.Fatal("transient failure lost durable verification requirement or credentials")
	}
	if _, _, _, err := MembershipClient(t.Context(), f.client, restored, io.Discard, nil, store.coordinate); err == nil {
		t.Fatal("reload bypassed current userinfo after transient failure")
	}
	f.status = 0
	if _, _, _, err := MembershipClient(t.Context(), f.client, restored, io.Discard, nil, store.coordinate); err != nil {
		t.Fatal(err)
	}
	if f.identity.rootRefresh.Load() != 1 || f.calls.Load() != 3 {
		t.Fatal("userinfo recovery rotated root again or used stale claims")
	}
}

func checkRootUserInfoFailure(t *testing.T, kind string) {
	t.Helper()
	f := newUserInfoFixture(t)
	f.identity.idExpiry = time.Now().Add(-time.Hour)
	root := f.identity.root(t)
	root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	switch kind {
	case "signature":
		f.access += "invalid"
	case "audience":
		f.access = signMembershipClaims(t, f.identity, map[string]any{"iss": root.Options.Issuer, "aud": "other", "sub": "user", "exp": time.Now().Add(time.Hour).Unix()})
	case "access subject":
		f.access = signMembershipClaims(t, f.identity, map[string]any{"iss": root.Options.Issuer, "aud": "fctl", "sub": "other", "exp": time.Now().Add(time.Hour).Unix()})
	case "userinfo subject":
		f.wrongSubject = true
	case "expired access":
		f.access = signMembershipClaims(t, f.identity, map[string]any{"iss": root.Options.Issuer, "aud": "fctl", "sub": "user", "exp": time.Now().Add(-time.Hour).Unix()})
	case "cross origin":
		f.endpoint = "https://other.invalid/userinfo"
	case "cancellation":
		f.wait = true
	}
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	store := newCoordinatedStore(root)
	_, _, _, err := MembershipClient(ctx, f.client, copyRoot(root), io.Discard, nil, store.coordinate)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unverified userinfo accepted or leaked credentials: %v", err)
	}
	if kind == "cancellation" && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("userinfo ignored cancellation")
	}
	if kind != "cross origin" && store.snapshot().MembershipToken.RefreshToken != "rotated-root-refresh" {
		t.Fatal("userinfo failure lost consumed refresh rotation")
	}
	if len(store.snapshot().Targets) != 0 {
		t.Fatal("userinfo failure persisted a target")
	}
}
