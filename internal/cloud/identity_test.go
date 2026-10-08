package cloud

import (
	"bytes"
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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type identityFixture struct {
	f                         *fixture
	client                    *http.Client
	accesses                  []organizationAccess
	mu                        sync.Mutex
	forms                     []url.Values
	used                      map[string]bool
	rootRefresh, childRefresh atomic.Int32
	invalidID, omitRefreshID  bool
	verification              string
	userCode                  string
	idExpiry                  time.Time
	refreshInvalidID          bool
	scopedID                  string
	childRefreshID            string
}

func newIdentityFixture(t *testing.T) *identityFixture {
	t.Helper()
	f := newFixture(t)
	fixture := &identityFixture{f: f, used: make(map[string]bool), accesses: []organizationAccess{{ID: "org", Stacks: []stackAccess{{ID: "stack", URI: f.server.URL + "/stack", Scopes: []string{"stack:Read", "stack:Write"}}}}}}
	fixture.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/membership/device" && r.URL.Path != "/membership/token" {
			return baseTransport(f.server.Client()).RoundTrip(r)
		}
		return fixture.authResponse(t, r), nil
	})}
	return fixture
}

func (f *identityFixture) signedID(t *testing.T) string {
	t.Helper()
	issuer := f.f.options.Issuer
	if f.f.identityIssuer != "" {
		issuer = f.f.identityIssuer
	}
	expiry := time.Now().Add(time.Hour)
	if !f.idExpiry.IsZero() {
		expiry = f.idExpiry
	}
	claims := map[string]any{"iss": issuer, "aud": f.f.audience, "sub": "user", "iat": time.Now().Unix(), "exp": expiry.Unix(), "org": f.accesses}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"key"}`))
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	hash := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.f.key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	raw := unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	if f.invalidID {
		raw += "invalid"
	}
	return raw
}

func (f *identityFixture) root(t *testing.T) *Session {
	t.Helper()
	return &Session{Options: Options{Issuer: f.f.options.Issuer, ClientID: "fctl"}, IDToken: f.signedID(t), MembershipToken: &oauth2.Token{AccessToken: "root-secret", RefreshToken: "root-refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}}
}

func (f *identityFixture) authResponse(t *testing.T, r *http.Request) *http.Response {
	t.Helper()
	recorder := httptest.NewRecorder()
	r.Body = http.MaxBytesReader(recorder, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		t.Error(err)
		recorder.WriteHeader(400)
		return recorder.Result()
	}
	clientID, _, ok := r.BasicAuth()
	if !ok || clientID != "fctl" {
		t.Error("missing public client authentication")
	}
	f.mu.Lock()
	f.forms = append(f.forms, r.Form)
	f.mu.Unlock()
	if r.URL.Path == "/membership/device" {
		code := "root-code"
		if r.Form.Get("resource") != "" {
			code = r.Form.Get("resource")
		}
		verification, userCode := f.verification, f.userCode
		if verification == "" {
			verification = f.f.server.URL + "/verify"
		}
		if userCode == "" {
			userCode = "ABCD"
		}
		writeJSON(t, recorder, map[string]any{"device_code": code, "user_code": userCode, "verification_uri": verification, "expires_in": 600, "interval": 1})
		return recorder.Result()
	}
	if r.Form.Get("grant_type") == "refresh_token" {
		return f.refreshResponse(t, recorder, r)
	}
	if r.Form.Has("scope") {
		recorder.WriteHeader(http.StatusBadRequest)
		writeJSON(t, recorder, map[string]string{"error": "server_error", "error_description": "requested scope not granted"})
		return recorder.Result()
	}
	access, refresh := "membership-secret", "refresh-secret"
	if r.Form.Get("device_code") == "root-code" {
		access, refresh = "root-secret", "root-refresh"
	}
	raw := f.signedID(t)
	if access == "membership-secret" && f.scopedID != "" {
		raw = f.scopedID
	}
	writeJSON(t, recorder, map[string]any{"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_in": 3600, "id_token": raw})
	return recorder.Result()
}

func (f *identityFixture) refreshResponse(t *testing.T, w *httptest.ResponseRecorder, r *http.Request) *http.Response {
	t.Helper()
	refresh := r.Form.Get("refresh_token")
	f.mu.Lock()
	used := f.used[refresh]
	f.used[refresh] = true
	f.mu.Unlock()
	if used {
		w.WriteHeader(400)
		writeJSON(t, w, map[string]string{"error": "invalid_grant", "error_description": "root-secret refresh-secret"})
		return w.Result()
	}
	access, rotated := "refreshed-secret", "rotated-secret"
	if refresh == "root-refresh" {
		f.rootRefresh.Add(1)
		access, rotated = "refreshed-root-secret", "rotated-root-refresh"
	} else {
		f.childRefresh.Add(1)
	}
	result := map[string]any{"access_token": access, "refresh_token": rotated, "token_type": "Bearer", "expires_in": 3600}
	if !f.omitRefreshID {
		raw := f.signedID(t)
		if refresh != "root-refresh" && f.childRefreshID != "" {
			raw = f.childRefreshID
		}
		if f.refreshInvalidID {
			raw += "invalid"
		}
		result["id_token"] = raw
	}
	writeJSON(t, w, result)
	return w.Result()
}

func TestIdentityDefaultOptions(t *testing.T) {
	options, err := identityOptions(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if options.Issuer != "https://app.formance.cloud/api" || options.ClientID != "fctl" || options.Organization != "" || options.Stack != "" {
		t.Fatalf("incorrect root defaults: %+v", options)
	}
	if _, err := identityOptions(Options{Organization: "org"}); err == nil {
		t.Fatal("identity login selected an organization")
	}
}

func TestLoginIdentityFormsAndBrowser(t *testing.T) {
	for _, kind := range []string{"zero access", "multiple accesses", "browser failure", "browser deadline", "browser canceled"} {
		t.Run(kind, func(t *testing.T) { t.Parallel(); checkIdentityLogin(t, kind) })
	}
}

func checkIdentityLogin(t *testing.T, kind string) {
	t.Helper()
	fixture := newIdentityFixture(t)
	switch kind {
	case "zero access":
		fixture.accesses = nil
	case "multiple accesses":
		fixture.accesses[0].Stacks = append(fixture.accesses[0].Stacks, stackAccess{ID: "other", URI: fixture.f.server.URL + "/stack", Scopes: []string{"stack:Read"}})
	}
	var out bytes.Buffer
	calls := 0
	session, err := LoginIdentity(t.Context(), fixture.client, Options{Issuer: fixture.f.options.Issuer, Organization: "one-off-org", Stack: "one-off-stack"}, &out, func(_ context.Context, uri string) error {
		calls++
		parsed, err := url.Parse(uri)
		if err != nil {
			return err
		}
		if parsed.Query().Get("user_code") != "ABCD" {
			t.Error("browser URI omitted user code")
		}
		if kind == "browser deadline" {
			return context.DeadlineExceeded
		}
		if kind == "browser canceled" {
			return context.Canceled
		}
		return errors.New("browser-secret")
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !strings.Contains(out.String(), "ABCD") || strings.Contains(out.String(), "secret") {
		t.Fatal("incorrect browser fallback or leaked credentials")
	}
	if session.Options.Organization != "" || session.Options.Stack != "" || session.StackURL != "" {
		t.Fatal("identity login selected a target")
	}
	assertIdentityForms(t, fixture.forms)
}

func assertIdentityForms(t *testing.T, forms []url.Values) {
	t.Helper()
	if len(forms) != 2 {
		t.Fatalf("unexpected number of root auth requests: %d", len(forms))
	}
	for _, form := range forms {
		if form.Get("organization_id") != "" || form.Get("resource") != "" {
			t.Fatal("root authorization requested a scoped or incomplete token")
		}
	}
	if forms[0].Get("scope") != "openid offline_access accesses on_behalf" || forms[1].Has("scope") {
		t.Fatal("root authorization scopes missing or repeated during polling")
	}
	if forms[0].Get("prompt") != "no-org" {
		t.Fatal("identity login omitted no-org prompt")
	}
}

func TestLoginIdentityRejectsUnverifiedClaims(t *testing.T) {
	for _, kind := range []string{"signature", "audience", "issuer", "expired", "invalid organization"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			fixture := newIdentityFixture(t)
			switch kind {
			case "signature":
				fixture.invalidID = true
			case "audience":
				fixture.f.audience = "other"
			case "issuer":
				fixture.f.identityIssuer = "https://other.example"
			case "expired":
				fixture.idExpiry = time.Now().Add(-time.Hour)
			case "invalid organization":
				fixture.accesses[0].ID = "org\nsecret"
			}
			_, err := LoginIdentity(t.Context(), fixture.client, Options{Issuer: fixture.f.options.Issuer}, io.Discard, nil)
			if err == nil || strings.Contains(err.Error(), "secret") {
				t.Fatalf("unsafe identity verification: %v", err)
			}
		})
	}
}

func TestTargetResolution(t *testing.T) {
	fixture := newIdentityFixture(t)
	options := fixture.root(t).Options
	claims := identityClaims{Organizations: []organizationAccess{
		{ID: "org", Stacks: []stackAccess{{ID: "first", URI: fixture.f.server.URL, Scopes: []string{"stack:Read"}}, {ID: "second", URI: fixture.f.server.URL, Scopes: []string{"stack:Write"}}}},
		{ID: "other", Stacks: []stackAccess{{ID: "second", URI: fixture.f.server.URL, Scopes: []string{"stack:Read"}}}},
	}}
	for _, test := range []struct {
		name    string
		desired Options
		valid   bool
		key     string
	}{
		{"ambiguous all", Options{}, false, ""}, {"ambiguous organization", Options{Organization: "org"}, false, ""},
		{"ambiguous stack", Options{Stack: "second"}, false, ""}, {"unique stack", Options{Stack: "first"}, true, "org/first"},
		{"unique organization", Options{Organization: "other"}, true, "other/second"},
		{"explicit", Options{Organization: "org", Stack: "second"}, true, "org/second"},
		{"unknown", Options{Organization: "missing", Stack: "second"}, false, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			target, err := resolveTarget(options, claims, test.desired)
			if (err == nil) != test.valid {
				t.Fatalf("incorrect target selection: %v", err)
			}
			if test.valid && targetKey(target.Options) != test.key {
				t.Fatal("wrong target selected")
			}
			if err != nil && (!strings.Contains(err.Error(), "--organization") || !strings.Contains(err.Error(), "cloud stack list") || strings.Contains(err.Error(), "org/first")) {
				t.Fatal("target selection error must give a concise discovery command")
			}
		})
	}
}

func TestTargetSelectionErrorRemainsBounded(t *testing.T) {
	f := newIdentityFixture(t)
	options := f.root(t).Options
	claims := identityClaims{Organizations: []organizationAccess{{ID: "org"}}}
	for i := range 1000 {
		claims.Organizations[0].Stacks = append(claims.Organizations[0].Stacks, stackAccess{
			ID: "stack-" + strconv.Itoa(i), URI: f.f.server.URL, Scopes: []string{"stack:Read"},
		})
	}
	_, err := resolveTarget(options, claims, Options{})
	if err == nil || len(err.Error()) > 250 || strings.Count(err.Error(), "\n") > 2 || strings.Contains(err.Error(), "stack-0") {
		t.Fatalf("target selection must not dump the access catalog: %v", err)
	}
}

func TestClientForTargetScopedAuthorizationAndCache(t *testing.T) {
	fixture := newIdentityFixture(t)
	fixture.accesses[0].Stacks[0].Scopes = []string{"stack:Read", "unrelated"}
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	var out bytes.Buffer
	client, uri, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, &out, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		resp, err := client.Do(testRequest(t, http.MethodPost, uri+"/ledger", nil))
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{Organization: "org", Stack: "stack"}, io.Discard, nil, store.coordinate); err != nil {
		t.Fatal(err)
	}
	saved := store.snapshot()
	if saved.Options.Organization != "" || saved.Options.Stack != "" || saved.MembershipToken.AccessToken != "root-secret" {
		t.Fatal("one-off target flags overwrote root identity")
	}
	if saved.Targets["org/stack"] == nil || saved.Targets["org/stack"].StackToken == nil {
		t.Fatal("scoped session not cached in root")
	}
	if len(fixture.forms) != 2 || fixture.f.exchanges.Load() != 1 {
		t.Fatal("cached target repeated device flow or stack exchange")
	}
	assertScopedForms(t, fixture.forms, root.IDToken, "stack://org/stack|stack:Read")
}

func assertScopedForms(t *testing.T, forms []url.Values, id, resource string) {
	t.Helper()
	if forms[0].Get("id_token_hint") != id || forms[0].Get("organization_id") != "org" || forms[0].Get("resource") != resource {
		t.Fatal("incorrect scoped authorization parameters")
	}
	for _, form := range forms {
		if form.Get("resource") != resource {
			t.Fatal("scoped flow requested broad scopes or lost resource")
		}
	}
	if forms[0].Get("scope") != "openid offline_access accesses" || forms[1].Has("scope") {
		t.Fatal("scoped authorization scopes missing or repeated during polling")
	}
}

func TestConcurrentTargetAuthorization(t *testing.T) {
	fixture := newIdentityFixture(t)
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	runParallel(t, func() error {
		_, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
		return err
	})
	if len(fixture.forms) != 2 || fixture.f.exchanges.Load() != 1 || len(store.snapshot().Targets) != 1 {
		t.Fatal("concurrent clients duplicated scoped authorization")
	}
}

func TestIdentityAndScopedRefreshStaySeparate(t *testing.T) {
	fixture := newIdentityFixture(t)
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	if _, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.session.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	store.session.Targets["org/stack"].MembershipToken.Expiry = time.Now().Add(-time.Hour)
	store.session.Targets["org/stack"].StackToken = nil
	store.mu.Unlock()
	runParallel(t, func() error {
		_, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
		return err
	})
	saved := store.snapshot()
	if fixture.rootRefresh.Load() != 1 || fixture.childRefresh.Load() != 1 {
		t.Fatal("root/child refresh was omitted or repeated")
	}
	if saved.MembershipToken.RefreshToken != "rotated-root-refresh" || saved.Targets["org/stack"].MembershipToken.RefreshToken != "rotated-secret" {
		t.Fatal("rotated root or child credentials were lost")
	}
}

func TestIdentityCancellation(t *testing.T) {
	fixture := newIdentityFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err := LoginIdentity(ctx, fixture.client, Options{Issuer: fixture.f.options.Issuer}, io.Discard, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("identity login ignored cancellation: %v", err)
	}
	if fixture.f.exchanges.Load() != 0 {
		t.Fatal("identity token was sent to stack auth")
	}
}

func TestVerificationQueryAndUserCode(t *testing.T) {
	fixture := newIdentityFixture(t)
	fixture.verification = fixture.f.server.URL + "/verify?locale=fr&flow=identity&user_code=old"
	var out bytes.Buffer
	calls := 0
	_, err := LoginIdentity(t.Context(), fixture.client, Options{Issuer: fixture.f.options.Issuer}, &out, func(_ context.Context, raw string) error {
		calls++
		parsed, err := url.Parse(raw)
		if err != nil {
			return err
		}
		query := parsed.Query()
		if query.Get("locale") != "fr" || query.Get("flow") != "identity" || query.Get("user_code") != "ABCD" || len(query["user_code"]) != 1 {
			t.Error("browser query was lost or user code was duplicated")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !strings.Contains(out.String(), fixture.verification) {
		t.Fatal("verification instructions or callback were omitted")
	}
}

func TestDeviceInstructionsRejectUnsafeInput(t *testing.T) {
	for _, test := range []struct{ name, uri, code string }{
		{"credentials", "https://user:password@example.com/verify", "ABCD"},
		{"cleartext", "http://example.com/verify", "ABCD"},
		{"relative", "/verify", "ABCD"},
		{"URL control", "https://example.com/verify\nsecret", "ABCD"},
		{"code newline", "https://example.com/verify", "ABC\nsecret"},
		{"code tab", "https://example.com/verify", "ABC\tsecret"},
		{"code escape", "https://example.com/verify", "ABC\x1bsecret"},
		{"malformed query", "https://example.com/verify?x=%invalid", "ABCD"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var out bytes.Buffer
			opened := false
			device := &oauth2.DeviceAuthResponse{DeviceCode: "device-secret", UserCode: test.code, VerificationURI: test.uri, Interval: 1, Expiry: time.Now().Add(time.Minute)}
			err := displayDevice(t.Context(), device, &out, func(context.Context, string) error { opened = true; return nil })
			if err == nil || strings.Contains(err.Error(), "secret") || opened || out.Len() != 0 {
				t.Fatal("unsafe device input was displayed or opened")
			}
		})
	}
}

func TestRootRefreshOmittedIdentityUsesOnlyUnexpiredClaims(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(strconv.FormatBool(expired), func(t *testing.T) { checkOmittedIdentityRefresh(t, expired) })
	}
}

func checkOmittedIdentityRefresh(t *testing.T, expired bool) {
	t.Helper()
	fixture := newIdentityFixture(t)
	if expired {
		fixture.idExpiry = time.Now().Add(-time.Hour)
	}
	root := fixture.root(t)
	root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	fixture.omitRefreshID = true
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if expired && (err == nil || !strings.Contains(err.Error(), "log in again")) {
		t.Fatalf("expired identity was accepted: %v", err)
	}
	if !expired && err != nil {
		t.Fatal(err)
	}
	if store.snapshot().MembershipToken.RefreshToken != "rotated-root-refresh" {
		t.Fatal("omitted ID token lost root refresh rotation")
	}
}

func TestRootRefreshInvalidIdentityPreservesRotation(t *testing.T) {
	fixture := newIdentityFixture(t)
	root := fixture.root(t)
	root.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	fixture.refreshInvalidID = true
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe identity refresh error: %v", err)
	}
	saved := store.snapshot()
	if saved.MembershipToken.RefreshToken != "rotated-root-refresh" || saved.IDToken != "" || len(saved.Targets) != 0 {
		t.Fatal("invalid fresh identity replaced claims or lost rotated credentials")
	}
	restored := roundTripSession(t, saved)
	reopened := newCoordinatedStore(restored)
	_, _, err = ClientForTarget(t.Context(), fixture.client, copyRoot(restored), Options{}, io.Discard, nil, reopened.coordinate)
	if err == nil || !strings.Contains(err.Error(), "log in again") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("persisted invalid identity was accepted: %v", err)
	}
	if fixture.rootRefresh.Load() != 1 || len(fixture.forms) != 1 {
		t.Fatal("blocked identity retried refresh or started scoped authorization")
	}
	if fixture.f.exchanges.Load() != 0 {
		t.Fatal("root token was exchanged after invalid identity")
	}
}

func TestCachedTargetRenewsChangedPermissions(t *testing.T) {
	for _, test := range []struct {
		name          string
		before, after []string
	}{
		{"downgrade", []string{"stack:Read", "stack:Write"}, []string{"stack:Read"}},
		{"upgrade", []string{"stack:Read"}, []string{"stack:Read", "stack:Write"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			checkChangedPermissions(t, test.before, test.after)
		})
	}
}

func checkChangedPermissions(t *testing.T, before, after []string) {
	t.Helper()
	fixture := newIdentityFixture(t)
	fixture.accesses[0].Stacks[0].Scopes = before
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	client, uri, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	oldID := store.snapshot().Targets["org/stack"].IDToken
	fixture.accesses[0].Stacks[0].Scopes = after
	newID := fixture.signedID(t)
	store.mu.Lock()
	store.session.IDToken = newID
	store.revision++
	store.mu.Unlock()
	// Exercise the existing client's lazy token path as well as a new client.
	resp, err := client.Do(testRequest(t, http.MethodPost, uri+"/ledger", nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	runParallel(t, func() error {
		_, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
		return err
	})
	if len(fixture.forms) != 4 || fixture.f.exchanges.Load() != 2 {
		t.Fatal("permission change did not renew authorization exactly once")
	}
	assertScopedForms(t, fixture.forms[2:], newID, "stack://org/stack|"+strings.Join(after, " "))
	if store.snapshot().Targets["org/stack"].IDToken == oldID {
		t.Fatal("cached child retained obsolete signed permissions")
	}
}

func TestLogoutDuringTargetAuthorizationCannotRestoreRoot(t *testing.T) {
	fixture := newIdentityFixture(t)
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, func(context.Context, string) error {
		store.mu.Lock()
		defer store.mu.Unlock()
		store.session = nil
		store.revision++
		return nil
	}, store.coordinate)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("logout did not invalidate authentication save: %v", err)
	}
	if store.snapshot() != nil || fixture.f.exchanges.Load() != 0 {
		t.Fatal("in-flight target login restored a logged-out root")
	}
}

func TestTargetAuthorizationCancellation(t *testing.T) {
	fixture := newIdentityFixture(t)
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, _, err := ClientForTarget(ctx, fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("target device flow ignored cancellation: %v", err)
	}
	if len(store.snapshot().Targets) != 0 || fixture.f.exchanges.Load() != 0 {
		t.Fatal("canceled scoped authorization persisted a target or reached stack auth")
	}
}

func TestRootSaveFailureStopsCachedClient(t *testing.T) {
	fixture := newIdentityFixture(t)
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	client, uri, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	store.expire()
	store.failSave = true
	for range 2 {
		assertCoordinatedRequestFails(t, client, uri)
	}
	if fixture.rootRefresh.Load() != 1 || fixture.f.writes.Load() != 0 {
		t.Fatal("failed root CAS retried a consumed token or sent service writes")
	}
	if store.snapshot().MembershipToken.RefreshToken != "root-refresh" {
		t.Fatal("failed CAS overwrote the stored root")
	}
}

func TestRootTargetCacheJSONRoundTrip(t *testing.T) {
	fixture := newIdentityFixture(t)
	root := fixture.root(t)
	root.Targets = map[string]*Session{"org/stack": fixture.f.session(t)}
	restored := roundTripSession(t, root)
	if restored.Options.Organization != "" || restored.Targets["org/stack"].Options.Stack != "stack" || restored.Targets["org/stack"].MembershipToken.RefreshToken != "refresh-secret" {
		t.Fatal("serialized root lost its identity or scoped child session")
	}
}

func TestFreshTargetRejectsMismatchedSignedPermissions(t *testing.T) {
	fixture := newIdentityFixture(t)
	fixture.scopedID = fixture.signedID(t) // Validly signed, but still grants Write.
	fixture.accesses[0].Stacks[0].Scopes = []string{"stack:Read"}
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	_, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if err == nil || err.Error() != "scoped session permissions differ from root identity" {
		t.Fatalf("fresh authorization accepted obsolete signed scopes: %v", err)
	}
	if len(store.snapshot().Targets) != 0 || fixture.f.exchanges.Load() != 0 || len(fixture.forms) != 2 {
		t.Fatal("mismatched authorization was persisted, exchanged, or retried")
	}
}

func TestChildRefreshRejectsElevatedPermissionsAndPreservesRotation(t *testing.T) {
	fixture := newIdentityFixture(t)
	fixture.childRefreshID = fixture.signedID(t) // Read/Write, despite a Read-only root.
	fixture.accesses[0].Stacks[0].Scopes = []string{"stack:Read"}
	root := fixture.root(t)
	store := newCoordinatedStore(root)
	client, uri, err := ClientForTarget(t.Context(), fixture.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	child := store.session.Targets["org/stack"]
	child.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	// A supplied in-memory guard is not authority: prepare must replace it.
	child.allowedScopes = []string{"stack:Read", "stack:Write"}
	store.revision++
	store.mu.Unlock()
	for range 2 {
		assertCoordinatedRequestFails(t, client, uri)
	}
	assertBlockedChildRotation(t, fixture, store.snapshot())
	restored := roundTripSession(t, store.snapshot())
	reopened := newCoordinatedStore(restored)
	if _, _, err := ClientForTarget(t.Context(), fixture.client, copyRoot(restored), Options{}, io.Discard, nil, reopened.coordinate); err == nil {
		t.Fatal("a new client accepted the blocked refreshed child")
	}
	assertBlockedChildRotation(t, fixture, reopened.snapshot())
}

func assertBlockedChildRotation(t *testing.T, fixture *identityFixture, saved *Session) {
	t.Helper()
	child := saved.Targets["org/stack"]
	if child.MembershipToken.RefreshToken != "rotated-secret" || child.IDToken != "" || child.StackToken != nil {
		t.Fatal("failed child verification lost rotation or retained usable identity")
	}
	if fixture.childRefresh.Load() != 1 || fixture.rootRefresh.Load() != 0 || fixture.f.exchanges.Load() != 1 || fixture.f.writes.Load() != 0 || len(fixture.forms) != 3 {
		t.Fatal("rejected refresh retried, authorized, exchanged, or reached the service")
	}
}
