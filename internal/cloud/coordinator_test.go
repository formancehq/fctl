package cloud

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// The authentication lock and store lock are deliberately separate, matching
// the parent's lock contract. Each callback save carries its loaded revision.
type coordinatedStore struct {
	auth      chan struct{}
	mu        sync.Mutex
	session   *Session
	revision  int
	saves     int
	callbacks int
	failSave  bool
}

func cloneSession(session *Session) *Session { return copyRoot(session) }

func newCoordinatedStore(session *Session) *coordinatedStore {
	return &coordinatedStore{auth: make(chan struct{}, 1), session: cloneSession(session)}
}

func (s *coordinatedStore) coordinate(ctx context.Context, work func(*Session, func(*Session) error) (*oauth2.Token, error)) (*oauth2.Token, error) {
	select {
	case s.auth <- struct{}{}:
		defer func() { <-s.auth }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	s.mu.Lock()
	current, revision := cloneSession(s.session), s.revision
	s.callbacks++
	s.mu.Unlock()
	return work(current, func(next *Session) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.saves++
		if s.failSave || revision != s.revision {
			return errors.New("revision changed")
		}
		s.session = cloneSession(next)
		s.revision++
		revision = s.revision
		return nil
	})
}

func (s *coordinatedStore) expire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	s.session.StackToken = nil
	s.revision++
}

func (s *coordinatedStore) snapshot() *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneSession(s.session)
}

// The provider consumes refresh-secret exactly once. A second use fails as a
// real rotating-refresh provider would; successful refresh returns a new token.
func rotatingClient(t *testing.T, f *fixture) *http.Client {
	t.Helper()
	base := f.server.Client()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/membership/token" {
			return baseTransport(base).RoundTrip(r)
		}
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		recorder := httptest.NewRecorder()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-secret" || f.refreshes.Add(1) != 1 {
			recorder.WriteHeader(http.StatusBadRequest)
			writeJSON(t, recorder, map[string]string{"error": "invalid_grant"})
		} else {
			writeJSON(t, recorder, map[string]any{"access_token": "refreshed-secret", "refresh_token": "rotated-secret", "token_type": "Bearer", "expires_in": 3600})
		}
		return recorder.Result(), nil
	})}
}

func runParallel(t *testing.T, work func() error) {
	t.Helper()
	start := make(chan struct{})
	failures := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() { <-start; failures <- work() })
	}
	close(start)
	workers.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCoordinatorSerializesEagerRefresh(t *testing.T) {
	f := newFixture(t)
	seed := f.session(t)
	seed.MembershipToken.Expiry = time.Now().Add(-time.Hour)
	store := newCoordinatedStore(seed)
	base := rotatingClient(t, f)
	runParallel(t, func() error {
		_, _, err := Client(t.Context(), base, cloneSession(seed), nil, store.coordinate)
		return err
	})
	if f.refreshes.Load() != 1 || f.exchanges.Load() != 1 || store.saves != 1 {
		t.Fatal("concurrent constructors repeated rotation or exchange")
	}
	if store.snapshot().MembershipToken.RefreshToken != "rotated-secret" {
		t.Fatal("disk retained the consumed refresh token")
	}
}

func TestCoordinatorSerializesLazyRefresh(t *testing.T) {
	f := newFixture(t)
	seed := f.session(t)
	seed.StackToken = &oauth2.Token{AccessToken: "stack-secret", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
	store := newCoordinatedStore(seed)
	base := rotatingClient(t, f)
	var clients []*http.Client
	for range 2 {
		client, _, err := Client(t.Context(), base, cloneSession(seed), nil, store.coordinate)
		if err != nil {
			t.Fatal(err)
		}
		clients = append(clients, client)
	}
	store.expire()
	start := make(chan struct{})
	failures := make(chan error, 2)
	var workers sync.WaitGroup
	for _, client := range clients {
		workers.Go(func() {
			<-start
			resp, err := client.Do(testRequest(t, http.MethodPost, seed.StackURL+"/ledger", nil))
			if err != nil {
				failures <- err
				return
			}
			failures <- resp.Body.Close()
		})
	}
	close(start)
	workers.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.refreshes.Load() != 1 || f.exchanges.Load() != 1 || store.saves != 1 || f.writes.Load() != 2 {
		t.Fatal("lazy authentication did not share the newly rotated tokens")
	}
	if store.callbacks != 4 || store.snapshot().MembershipToken.RefreshToken != "rotated-secret" {
		t.Fatal("lazy calls bypassed coordination or rewrote a consumed refresh token")
	}
}

func TestCoordinatorSaveFailureStopsReload(t *testing.T) {
	f := newFixture(t)
	seed := f.session(t)
	seed.StackToken = &oauth2.Token{AccessToken: "stack-secret", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
	store := newCoordinatedStore(seed)
	client, _, err := Client(t.Context(), rotatingClient(t, f), cloneSession(seed), nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	store.expire()
	store.failSave = true
	for range 2 {
		resp, err := client.Do(testRequest(t, http.MethodPost, seed.StackURL+"/ledger", nil))
		if resp != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
		}
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe save failure: %v", err)
		}
	}
	transport, ok := client.Transport.(*sessionTransport)
	if !ok {
		t.Fatal("unexpected transport")
	}
	if !transport.pendingSave || transport.state.MembershipToken.RefreshToken != "rotated-secret" {
		t.Fatal("reload overwrote pending rotated credentials")
	}
	if f.refreshes.Load() != 1 || store.saves != 1 || f.writes.Load() != 0 {
		t.Fatal("save failure retried a consumed token or sent a service write")
	}
}

func TestCoordinatorValidatesReloadedSession(t *testing.T) {
	for _, kind := range []string{"missing", "options", "signature", "audience", "scope", "stackURL", "issuer", "stack", "signedURI"} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t)
			seed := f.session(t)
			store := newCoordinatedStore(seed)
			client, _, err := Client(t.Context(), f.server.Client(), cloneSession(seed), nil, store.coordinate)
			if err != nil {
				t.Fatal(err)
			}
			changeCoordinatedSession(t, f, store, kind)
			assertCoordinatedRequestFails(t, client, seed.StackURL)
			if f.refreshes.Load() != 0 || f.exchanges.Load() != 1 || f.writes.Load() != 0 {
				t.Fatal("invalid identity triggered authentication or service operations")
			}
		})
	}
}

func assertCoordinatedRequestFails(t *testing.T, client *http.Client, stackURL string) {
	t.Helper()
	resp, err := client.Do(testRequest(t, http.MethodPost, stackURL+"/ledger", nil))
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if err == nil {
		t.Fatal("accepted invalid reloaded session")
	}
}

func changeCoordinatedSession(t *testing.T, f *fixture, store *coordinatedStore, kind string) {
	t.Helper()
	switch kind {
	case "missing":
		store.session = nil
	case "options":
		store.session.Options.Organization = "other"
	case "signature":
		store.session.IDToken = "invalid"
	case "audience":
		f.audience = "other"
		store.session.IDToken = f.id(t)
	case "scope":
		f.noScopes = true
		store.session.IDToken = f.id(t)
	case "stackURL":
		store.session.StackURL = "https://other.example"
	case "issuer":
		f.identityIssuer = "https://other.example"
		store.session.IDToken = f.id(t)
	case "stack":
		f.stack = "other"
		store.session.IDToken = f.id(t)
	case "signedURI":
		f.claimsURI = f.server.URL + "/other"
		store.session.IDToken = f.id(t)
		store.session.StackURL = f.claimsURI
	}
}

func TestCoordinatorWaitCancellation(t *testing.T) {
	f := newFixture(t)
	store := newCoordinatedStore(f.session(t))
	client, stackURL, err := Client(t.Context(), f.server.Client(), store.snapshot(), nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate another process holding the authentication lock.
	store.auth <- struct{}{}
	defer func() { <-store.auth }()
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stackURL+"/ledger", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("coordinator ignored request cancellation: %v", err)
	}
	if f.writes.Load() != 0 {
		t.Fatal("service write escaped cancellation")
	}
}
