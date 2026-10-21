package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/auth"
)

func TestUpdatePreservesOmittedOptions(t *testing.T) {
	const current = `{"data":{"id":"client-id","name":"Original","description":"Keep me","public":true,"trusted":true,"redirectUris":["https://example.com"],"postLogoutRedirectUris":["https://example.com/logout"],"scopes":["auth:read"],"metadata":{"id":"9007199254740993"},"secrets":[{"id":"secret","clear":"never-send"}],"serverOnly":9007199254740993}}`
	for _, tc := range []struct {
		name    string
		flags   map[string]string
		changed map[string]bool
		body    string
		changes string
	}{
		{name: "omitted host defaults", flags: map[string]string{"description": "Changed", "public": "false", "trusted": "false", "client-scopes": "", "redirect-uri": "", "post-logout-redirect-uri": "", "name": ""}, changed: map[string]bool{"description": true}, changes: `{"description":"Changed"}`},
		{name: "explicit false", flags: map[string]string{"public": "false", "trusted": "false"}, changed: map[string]bool{"public": true, "trusted": true}, changes: `{"public":false,"trusted":false}`},
		{name: "direct SDK false", flags: map[string]string{"public": "false"}, changes: `{"public":false}`},
		{name: "explicit empty", flags: map[string]string{"description": "", "client-scopes": "", "redirect-uri": "", "post-logout-redirect-uri": ""}, changed: map[string]bool{"description": true, "client-scopes": true, "redirect-uri": true, "post-logout-redirect-uri": true}, changes: `{"description":"","scopes":[],"redirectUris":[],"postLogoutRedirectUris":[]}`},
		{name: "direct empty", flags: map[string]string{"description": ""}, changes: `{"description":""}`},
		{name: "rename flag", flags: map[string]string{"name": "Renamed"}, changes: `{"name":"Renamed"}`},
		{name: "false string is a value", flags: map[string]string{"description": "false"}, changed: map[string]bool{}, changes: `{"description":"false"}`},
		{name: "CSV update", flags: map[string]string{"client-scopes": "auth:write,ledger:write"}, changes: `{"scopes":["auth:write","ledger:write"]}`},
		{name: "body patch", body: `{"public":false,"redirectUris":[],"metadata":{}}`, changes: `{"public":false,"redirectUris":[],"metadata":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := updateFixture(t, current, tc.changes)
			req := request("auth clients update", "client-id")
			req.Flags, req.ChangedFlags, req.Endpoint = tc.flags, tc.changed, server.URL+"/api/auth"
			if tc.body != "" {
				req.Body = json.RawMessage(tc.body)
			}
			req = approve(req)
			before := cloneRequest(req)
			response, err := auth.New(server.Client()).Execute(t.Context(), req)
			if err != nil || calls.Load() != 2 || string(response.Data) != `{"data":{"quantity":9007199254740993}}` {
				t.Fatalf("result=%s; err=%v; calls=%d", response.Data, err, calls.Load())
			}
			assertRequestUnchanged(t, req, before)
		})
	}
}

func TestCancellationDuringUpdateReadPreventsPut(t *testing.T) {
	entered := make(chan struct{})
	var reads, writes atomic.Int32
	server := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		reads.Add(1)
		close(entered)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req := approve(request("auth clients update", "client"))
	req.Flags["description"], req.Endpoint = "changed", server.URL
	finished := make(chan error, 1)
	go func() {
		_, err := auth.New(server.Client()).Execute(ctx, req)
		finished <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("update never started its read")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("update did not stop after cancellation")
	}
	if reads.Load() != 1 || writes.Load() != 0 {
		t.Fatalf("reads = %d; writes = %d", reads.Load(), writes.Load())
	}
}
