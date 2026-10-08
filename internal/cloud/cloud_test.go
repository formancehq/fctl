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
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/oauth2"
)

type fixture struct {
	server                                 *httptest.Server
	key                                    *rsa.PrivateKey
	options                                Options
	audience, stack, organization          string
	signingKey                             *rsa.PrivateKey
	pollError, refreshError, exchangeError string
	refreshedID                            string
	omitID                                 bool
	deviceRedirect                         string
	tokenRedirect                          string
	claimsURI                              string
	deviceWait                             bool
	identityIssuer                         string
	noScopes                               bool
	pollResults                            []string
	waitPath                               string
	polls, refreshes, exchanges, writes    atomic.Int32
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{key: key, audience: "fctl", stack: "stack", organization: "org"}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, w, r) }))
	t.Cleanup(f.server.Close)
	f.options = Options{Issuer: f.server.URL + "/membership", Organization: "org", Stack: "stack"}
	return f
}
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Error(err)
	}
}
func (f *fixture) id(t *testing.T) string {
	t.Helper()
	uri := f.server.URL + "/stack"
	if f.claimsURI != "" {
		uri = f.claimsURI
	}
	issuer := f.options.Issuer
	if f.identityIssuer != "" {
		issuer = f.identityIssuer
	}
	scopes := []string{"stack:Read", "stack:Write"}
	if f.noScopes {
		scopes = []string{"organization:Read"}
	}
	claims := map[string]any{"iss": issuer, "aud": f.audience, "sub": "user", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "org": []any{map[string]any{"id": f.organization, "stacks": []any{map[string]any{"id": f.stack, "uri": uri, "scopes": scopes}}}}}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"key"}`))
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	hash := sha256.Sum256([]byte(unsigned))
	key := f.key
	if f.signingKey != nil {
		key = f.signingKey
	}
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
}
func (f *fixture) session(t *testing.T) *Session {
	return &Session{Options: f.options, IDToken: f.id(t), StackURL: f.server.URL + "/stack", MembershipToken: &oauth2.Token{AccessToken: "membership-secret", TokenType: "Bearer", RefreshToken: "refresh-secret", Expiry: time.Now().Add(time.Hour)}}
}
func (f *fixture) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if r.URL.Path == f.waitPath {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Error(err)
			return
		}
		<-r.Context().Done()
		return
	}
	switch r.URL.Path {
	case "/membership/.well-known/openid-configuration":
		writeJSON(t, w, map[string]any{"issuer": f.options.Issuer, "token_endpoint": f.server.URL + "/membership/token", "device_authorization_endpoint": f.server.URL + "/membership/device", "jwks_uri": f.server.URL + "/membership/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
	case "/membership/keys":
		writeJSON(t, w, map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "key", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(f.key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.key.E)).Bytes())}}})
	case "/membership/device":
		f.serveDevice(t, w, r)
	case "/membership/token":
		f.serveMembershipToken(t, w, r)
	case "/stack/api/auth/.well-known/openid-configuration":
		writeJSON(t, w, map[string]any{"issuer": f.server.URL + "/stack/api/auth", "token_endpoint": f.server.URL + "/stack/api/auth/token"})
	case "/stack/api/auth/token":
		f.serveExchange(t, w, r)
	case "/stack/ledger":
		f.writes.Add(1)
		if r.Header.Get("Authorization") != "Bearer stack-secret" {
			t.Error("service received wrong bearer")
		}
		w.WriteHeader(http.StatusUnauthorized)
	default:
		http.NotFound(w, r)
	}
}

func TestLoginAndClient(t *testing.T) {
	f := newFixture(t)
	var out bytes.Buffer
	session, err := Login(t.Context(), f.server.Client(), f.options, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ABCD") || strings.Contains(out.String(), "secret") {
		t.Fatalf("unexpected device output: %s", out.String())
	}
	restored := roundTripSession(t, session)
	saves := 0
	client, stackURL, err := Client(t.Context(), f.server.Client(), restored, func(s *Session) error {
		saves++
		if s.StackToken == nil {
			t.Error("missing saved stack token")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if stackURL != f.server.URL+"/stack" || saves != 1 {
		t.Fatal("client did not exchange and persist")
	}
	for range 2 {
		req := testRequest(t, http.MethodPost, stackURL+"/ledger", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if f.writes.Load() != 2 || f.exchanges.Load() != 1 {
		t.Fatal("service writes were retried or stack token was not cached")
	}
}

func TestLoginRejectsIdentity(t *testing.T) {
	for _, kind := range []string{"signature", "audience", "organization", "stack", "missingID", "unsafeURI", "issuer", "scope"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			switch kind {
			case "signature":
				key, err := rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					t.Fatal(err)
				}
				f.signingKey = key
			case "audience":
				f.audience = "other"
			case "organization":
				f.organization = "other"
			case "stack":
				f.stack = "other"
			case "issuer":
				f.identityIssuer = "https://other.example"
			case "scope":
				f.noScopes = true
			case "missingID":
				f.omitID = true
			case "unsafeURI":
				f.claimsURI = "https://user:secret@example.com"
			}
			if _, err := Login(t.Context(), f.server.Client(), f.options, io.Discard); err == nil {
				t.Fatal("accepted invalid identity")
			} else if strings.Contains(err.Error(), "secret") {
				t.Fatal("leaked token")
			}
			if f.exchanges.Load() != 0 {
				t.Fatal("exchanged an unverified identity")
			}
		})
	}
}

func TestRefreshPreservesRefreshToken(t *testing.T) {
	f := newFixture(t)
	session := f.session(t)
	session.MembershipToken.Expiry = time.Now().Add(10 * time.Second)
	session.StackToken = &oauth2.Token{AccessToken: "old-stack", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
	saves := 0
	_, _, err := Client(t.Context(), f.server.Client(), session, func(s *Session) error { saves++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if f.refreshes.Load() != 1 || f.exchanges.Load() != 1 || saves != 1 || session.MembershipToken.RefreshToken != "refresh-secret" || session.MembershipToken.AccessToken != "refreshed-secret" {
		t.Fatal("refresh or persistence failed")
	}
}

func TestSafeTokenErrors(t *testing.T) {
	for _, kind := range []string{"device", "refresh", "exchange"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			secret := `{"error":"access_denied","error_description":"membership-secret refresh-secret device-secret"}`
			var err error
			switch kind {
			case "device":
				f.pollError = secret
				_, err = Login(t.Context(), f.server.Client(), f.options, io.Discard)
			case "refresh":
				f.refreshError = secret
				s := f.session(t)
				s.MembershipToken.Expiry = time.Now().Add(-time.Minute)
				_, _, err = Client(t.Context(), f.server.Client(), s, nil)
			case "exchange":
				f.exchangeError = secret
				_, _, err = Client(t.Context(), f.server.Client(), f.session(t), nil)
			}
			if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "description") {
				t.Fatalf("unsafe error: %v", err)
			}
		})
	}
}

func TestLoginCancellation(t *testing.T) {
	for _, phase := range []string{"device", "poll"} {
		t.Run(phase, func(t *testing.T) {
			f := newFixture(t)
			f.deviceWait = phase == "device"
			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()
			start := time.Now()
			_, err := Login(ctx, f.server.Client(), f.options, io.Discard)
			if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
				t.Fatalf("cancellation failed: %v", err)
			}
		})
	}
}

func TestDeviceRedirect(t *testing.T) {
	f := newFixture(t)
	var reached atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1) }))
	defer other.Close()
	f.deviceRedirect = other.URL
	if _, err := Login(t.Context(), f.server.Client(), f.options, io.Discard); err == nil {
		t.Fatal("accepted device redirect")
	}
	if reached.Load() != 0 {
		t.Fatal("redirect leaked device request")
	}
}

func TestTokenRedirect(t *testing.T) {
	f := newFixture(t)
	var reached atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1) }))
	defer other.Close()
	f.tokenRedirect = other.URL
	if _, err := Login(t.Context(), f.server.Client(), f.options, io.Discard); err == nil {
		t.Fatal("accepted token redirect")
	}
	if reached.Load() != 0 || f.polls.Load() != 1 {
		t.Fatal("token request redirected or retried")
	}
}

func TestDataPlaneOrigin(t *testing.T) {
	f := newFixture(t)
	client, _, err := Client(t.Context(), f.server.Client(), f.session(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	var reached atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1) }))
	defer other.Close()
	req := testRequest(t, http.MethodPost, other.URL, nil)
	resp, err := client.Do(req)
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}
	if err == nil {
		t.Fatal("accepted cross-origin service request")
	}
	if reached.Load() != 0 {
		t.Fatal("leaked service token")
	}
}

func TestClientRejectsChangedStackURL(t *testing.T) {
	f := newFixture(t)
	session := f.session(t)
	session.StackURL = "https://other.example"
	if _, _, err := Client(t.Context(), f.server.Client(), session, nil); err == nil {
		t.Fatal("trusted unsigned stack URI")
	}
	if f.exchanges.Load() != 0 {
		t.Fatal("exchanged for unsigned stack URI")
	}
}

func TestRefreshRejectsNewIdentity(t *testing.T) {
	f := newFixture(t)
	session := f.session(t)
	session.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	f.audience = "other"
	f.refreshedID = f.id(t)
	if _, _, err := Client(t.Context(), f.server.Client(), session, nil); err == nil {
		t.Fatal("accepted invalid refreshed ID token")
	}
	if f.exchanges.Load() != 0 {
		t.Fatal("exchanged after invalid refreshed identity")
	}
}

func TestRefreshSavedOnExchangeFailure(t *testing.T) {
	f := newFixture(t)
	session := f.session(t)
	session.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	f.exchangeError = "refreshed-secret"
	saves := 0
	_, _, err := Client(t.Context(), f.server.Client(), session, func(s *Session) error { saves++; return nil })
	if err == nil || saves != 1 || session.MembershipToken.AccessToken != "refreshed-secret" {
		t.Fatal("rotated refresh credentials were lost on exchange failure")
	}
}

func TestSaveFailureRetried(t *testing.T) {
	f := newFixture(t)
	session := f.session(t)
	calls := 0
	client, uri, err := Client(t.Context(), f.server.Client(), session, func(s *Session) error {
		calls++
		if calls == 2 {
			return errors.New("refresh-secret")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := client.Transport.(*sessionTransport)
	if !ok {
		t.Fatal("unexpected cloud transport")
	}
	transport.mu.Lock()
	transport.state.StackToken.Expiry = time.Now().Add(-time.Hour)
	transport.mu.Unlock()
	req := testRequest(t, http.MethodGet, uri+"/ledger", nil)
	resp, err := client.Do(req)
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe save failure: %v", err)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if calls != 3 || f.writes.Load() != 1 {
		t.Fatal("failed save was not retried before service operation")
	}
}

func TestTokenBounds(t *testing.T) {
	now := time.Now()
	for _, token := range []*oauth2.Token{nil, {AccessToken: "x"}, {AccessToken: "x", Expiry: now.Add(-time.Second)}, {AccessToken: "x", TokenType: "MAC", Expiry: now.Add(time.Hour)}, {AccessToken: "x\n", Expiry: now.Add(time.Hour)}} {
		if err := boundToken(token, now); err == nil {
			t.Fatal("accepted invalid token")
		}
	}
	token := &oauth2.Token{AccessToken: "x", Expiry: now.Add(365 * 24 * time.Hour)}
	if err := boundToken(token, now); err != nil {
		t.Fatal(err)
	}
	if token.Expiry != now.Add(maxTokenLifetime) || token.TokenType != "Bearer" {
		t.Fatal("token lifetime was not bounded")
	}
}

func TestClientCancellation(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	client, uri, err := Client(ctx, f.server.Client(), f.session(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	req := testRequest(t, http.MethodPost, uri+"/ledger", nil)
	resp, err := client.Do(req)
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("client lifetime cancellation failed: %v", err)
	}
	if f.writes.Load() != 0 {
		t.Fatal("wrote after client cancellation")
	}
}

func (f *fixture) serveDevice(t *testing.T, w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if f.deviceWait {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Error(err)
			return
		}
		<-r.Context().Done()
		return
	}
	if f.deviceRedirect != "" {
		http.Redirect(w, r, f.deviceRedirect, http.StatusTemporaryRedirect)
		return
	}
	if err := r.ParseForm(); err != nil {
		t.Error(err)
		return
	}
	id, _, ok := r.BasicAuth()
	if !ok || id != "fctl" {
		t.Error("missing public client Basic authentication")
	}
	if r.Form.Get("client_id") != "fctl" || r.Form.Get("organization_id") != "org" || r.Form.Get("resource") != f.options.resource() || r.Form.Get("scope") != "openid offline_access" {
		t.Errorf("incorrect device parameters: %v", r.Form)
	}
	writeJSON(t, w, map[string]any{"device_code": "device-secret", "user_code": "ABCD", "verification_uri": f.server.URL + "/verify", "expires_in": 600, "interval": 1})
}

func (f *fixture) serveMembershipToken(t *testing.T, w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		t.Error(err)
		return
	}
	id, _, ok := r.BasicAuth()
	if !ok || id != "fctl" {
		t.Error("missing token client authentication")
	}
	grant := r.Form.Get("grant_type")
	if grant == "refresh_token" {
		f.serveRefresh(t, w, r)
		return
	}
	poll := f.polls.Add(1)
	if int(poll) <= len(f.pollResults) && f.pollResults[poll-1] != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(t, w, map[string]string{"error": f.pollResults[poll-1]})
		return
	}
	if grant != "urn:ietf:params:oauth:grant-type:device_code" || r.Form.Get("resource") != f.options.resource() || r.Form.Get("device_code") != "device-secret" {
		t.Error("incorrect polling grant or resource")
	}
	if f.tokenRedirect != "" {
		http.Redirect(w, r, f.tokenRedirect, http.StatusTemporaryRedirect)
		return
	}
	if f.pollError != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		if _, err := io.WriteString(w, f.pollError); err != nil {
			t.Error(err)
		}
		return
	}
	result := map[string]any{"access_token": "membership-secret", "refresh_token": "refresh-secret", "token_type": "Bearer", "expires_in": 3600}
	if !f.omitID {
		result["id_token"] = f.id(t)
	}
	writeJSON(t, w, result)
}

func (f *fixture) serveRefresh(t *testing.T, w http.ResponseWriter, r *http.Request) {

	f.refreshes.Add(1)
	if r.Form.Get("refresh_token") != "refresh-secret" {
		t.Error("wrong refresh token")
	}
	if f.refreshError != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		if _, err := io.WriteString(w, f.refreshError); err != nil {
			t.Error(err)
		}
		return
	}
	result := map[string]any{"access_token": "refreshed-secret", "token_type": "Bearer", "expires_in": 3600}
	if f.refreshedID != "" {
		result["id_token"] = f.refreshedID
	}
	writeJSON(t, w, result)
}

func (f *fixture) serveExchange(t *testing.T, w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	f.exchanges.Add(1)
	if err := r.ParseForm(); err != nil {
		t.Error(err)
		return
	}
	if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" || r.Form.Get("scope") != "openid email" || (r.Form.Get("assertion") != "membership-secret" && r.Form.Get("assertion") != "refreshed-secret") {
		t.Error("incorrect stack exchange")
	}
	if f.exchangeError != "" {
		w.WriteHeader(401)
		if _, err := io.WriteString(w, f.exchangeError); err != nil {
			t.Error(err)
		}
		return
	}
	writeJSON(t, w, map[string]any{"access_token": "stack-secret", "token_type": "Bearer", "expires_in": 3600})
}

func testRequest(t *testing.T, method, uri string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, uri, body)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func roundTripSession(t *testing.T, session *Session) *Session {
	t.Helper()
	encoded, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	var restored Session
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	return &restored
}

// The OAuth library owns RFC 8628 polling. Fake time checks pending and slow_down
// without adding seconds of real waiting to the suite.
func TestPollingIntervals(t *testing.T) {
	f := newFixture(t)
	f.pollResults = []string{"authorization_pending", "slow_down", ""}
	synctest.Test(t, func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			f.serve(t, recorder, r)
			return recorder.Result(), nil
		})}
		start := time.Now()
		if _, err := Login(t.Context(), client, f.options, io.Discard); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != 8*time.Second || f.polls.Load() != 3 {
			t.Fatalf("incorrect polling schedule: %v, polls=%d", elapsed, f.polls.Load())
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestExchangeAndKeysCancellation(t *testing.T) {
	for _, path := range []string{"/membership/keys", "/stack/api/auth/token"} {
		t.Run(path, func(t *testing.T) {
			f := newFixture(t)
			f.waitPath = path
			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()
			_, _, err := Client(ctx, f.server.Client(), f.session(t), nil)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("authentication cancellation failed: %v", err)
			}
		})
	}
}
