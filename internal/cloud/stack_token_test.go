package cloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestStackAccessTokenRejectsOtherClients(t *testing.T) {
	for _, client := range []*http.Client{nil, {}, {Transport: &membershipTransport{}}} {
		if token, err := StackAccessToken(t.Context(), client); err == nil || token != "" {
			t.Fatal("a non-stack client exposed a token")
		}
	}
}

func TestStackAccessTokenUsesCoordinatedStackSession(t *testing.T) {
	f := newIdentityFixture(t)
	root := f.root(t)
	store := newCoordinatedStore(root)
	client, _, err := ClientForTarget(t.Context(), f.client, copyRoot(root), Options{}, io.Discard, nil, store.coordinate)
	if err != nil {
		t.Fatal(err)
	}
	before := store.callbacks
	token, err := StackAccessToken(t.Context(), client)
	if err != nil || token != "stack-secret" || store.callbacks != before+1 {
		t.Fatal("stack token bypassed coordinator or returned Membership credentials")
	}
	store.mu.Lock()
	store.session.Targets["org/stack"].MembershipToken.Expiry = root.MembershipToken.Expiry.Add(-maxTokenLifetime)
	store.session.Targets["org/stack"].StackToken = nil
	store.revision++
	store.mu.Unlock()
	if token, err := StackAccessToken(t.Context(), client); err != nil || token != "stack-secret" {
		t.Fatal("token helper failed to refresh the stack session")
	}
	if f.childRefresh.Load() != 1 || store.snapshot().Targets["org/stack"].MembershipToken.RefreshToken != "rotated-secret" || f.f.writes.Load() != 0 {
		t.Fatal("token refresh lost rotation or sent a service request")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if token, err := StackAccessToken(ctx, client); !errors.Is(err, context.Canceled) || token != "" {
		t.Fatal("token helper ignored cancellation")
	}
	store.mu.Lock()
	store.session = nil
	store.revision++
	store.mu.Unlock()
	if token, err := StackAccessToken(t.Context(), client); err == nil || token != "" {
		t.Fatal("token helper accepted a logged-out session")
	}
}
