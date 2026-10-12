package cloud

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestScopedStackProofUsesAccessJWT(t *testing.T) {
	for _, kind := range []string{"valid", "signature", "audience", "organization", "stack", "elevated", "subject", "expired"} {
		t.Run(kind, func(t *testing.T) { t.Parallel(); checkScopedStackProof(t, kind) })
	}
}

func TestCachedStackScopeChangeRejectsInvalidAccessProof(t *testing.T) {
	for _, kind := range []string{"signature", "audience", "organization", "stack", "subject"} {
		t.Run(kind, func(t *testing.T) { checkChangedScopeInvalidAccess(t, kind) })
	}
}

func checkChangedScopeInvalidAccess(t *testing.T, kind string) {
	t.Helper()
	f := newIdentityFixture(t)
	f.accesses[0].Stacks[0].Scopes = []string{"stack:Read", "stack:Write"}
	options := f.f.options
	options.ClientID = "fctl"
	uri := f.f.server.URL + "/stack"
	claims := map[string]any{"iss": options.Issuer, "aud": uri + "/api/auth", "sub": "user", "exp": time.Now().Add(time.Hour).Unix(), "organization_id": "org", "stack_id": "stack", "scope": "ledger:read ledger:write"}
	alterStackAccessClaims(claims, kind)
	access := signMembershipClaims(t, f, claims)
	if kind == "signature" {
		access += "invalid"
	}
	child := &Session{Options: options, StackURL: uri, IDToken: f.signedID(t), MembershipToken: &oauth2.Token{AccessToken: access, Expiry: time.Now().Add(time.Hour)}}
	f.accesses[0].Stacks[0].Scopes = []string{"stack:Read"}
	root := f.root(t)
	root.Targets = map[string]*Session{"org/stack": child}
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if err == nil || len(f.forms) != 0 || f.rootRefresh.Load() != 0 || f.f.exchanges.Load() != 0 {
		t.Fatal("invalid cached access proof became a scope cache miss and triggered authentication")
	}
}

func alterStackAccessClaims(claims map[string]any, kind string) {
	switch kind {
	case "audience":
		claims["aud"] = "other"
	case "organization":
		claims["organization_id"] = "other"
	case "stack":
		claims["stack_id"] = "other"
	case "subject":
		claims["sub"] = "other"
	}
}

func checkScopedStackProof(t *testing.T, kind string) {
	t.Helper()
	f := newIdentityFixture(t)
	provider, _, err := membership(t.Context(), f.client, Options{Issuer: f.f.options.Issuer, ClientID: "fctl"})
	if err != nil {
		t.Fatal(err)
	}
	target := targetAccess{Options: f.f.options, URI: f.f.server.URL + "/stack", Scopes: []string{"stack:Read"}, Subject: "user"}
	target.Options.ClientID = "fctl"
	claims := map[string]any{"iss": target.Options.Issuer, "aud": target.URI + "/api/auth", "sub": "user", "exp": time.Now().Add(time.Hour).Unix(), "organization_id": "org", "stack_id": "stack", "scope": "ledger:read auth:read"}
	switch kind {
	case "audience":
		claims["aud"] = "other"
	case "organization":
		claims["organization_id"] = "other"
	case "stack":
		claims["stack_id"] = "other"
	case "elevated":
		claims["scope"] = "ledger:read auth:read ledger:write"
	case "subject":
		claims["sub"] = "other"
	case "expired":
		claims["exp"] = time.Now().Add(-time.Hour).Unix()
	}
	access := signMembershipClaims(t, f, claims)
	if kind == "signature" {
		access += "invalid"
	}
	id := signMembershipClaims(t, f, map[string]any{"iss": target.Options.Issuer, "aud": "fctl", "sub": "user", "exp": time.Now().Add(time.Hour).Unix()})
	child := &Session{Options: target.Options, StackURL: target.URI, IDToken: id, MembershipToken: &oauth2.Token{AccessToken: access, TokenType: "Bearer", RefreshToken: "refresh", Expiry: time.Now().Add(time.Hour)}}
	matches, err := verifyTargetSession(t.Context(), provider, target, child, false, false)
	if (err == nil && matches) != (kind == "valid") {
		t.Fatal("scoped proof did not enforce signature, audience, target, subject, scopes and expiration")
	}
}

func TestOrganizationRefreshOmittedIDAcceptsOnlyFreshAccessProof(t *testing.T) {
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
	expiredID := signMembershipClaims(t, f.identity, map[string]any{"iss": root.Options.Issuer, "aud": "fctl", "sub": "user", "exp": time.Now().Add(-time.Hour).Unix()})
	store.mu.Lock()
	store.session.Organizations["org"].IDToken = expiredID
	store.session.Organizations["org"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	store.revision++
	store.mu.Unlock()
	base := f.client.Transport
	f.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		resp, err := base.RoundTrip(r)
		if err != nil || r.URL.Path != "/membership/token" || r.Form.Get("grant_type") != "refresh_token" {
			return resp, err
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			closeResponse(resp.Body)
			return nil, err
		}
		closeResponse(resp.Body)
		delete(body, "id_token")
		w := httptest.NewRecorder()
		writeJSON(t, w, body)
		return w.Result(), nil
	})
	if err := membershipRequest(t, client, issuer+"/organizations/org"); err != nil {
		t.Fatal(err)
	}
	if store.snapshot().Organizations["org"].MembershipToken.RefreshToken != "rotated-organization-refresh" || f.refreshes.Load() != 1 {
		t.Fatal("omitted ID token lost organization rotation")
	}
}
