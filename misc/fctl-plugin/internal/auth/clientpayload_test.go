package auth_test

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/auth"
)

func TestPayloadTypes(t *testing.T) {
	for _, field := range []string{
		`"name":null`, `"name":9`, `"name":" "`, `"description":null`, `"description":false`,
		`"public":"false"`, `"trusted":null`, `"redirectUris":{}`, `"redirectUris":[null]`,
		`"scopes":[9]`, `"postLogoutRedirectUris":null`, `"metadata":[]`, `"metadata":{"n":9007199254740993}`, `"metadata":{"n":null}`,
	} {
		t.Run(field, func(t *testing.T) {
			var calls atomic.Int32
			server := fixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			req := approve(request("auth clients update", "client"))
			req.Body, req.Endpoint = json.RawMessage("{"+field+"}"), server.URL
			if _, err := auth.New(server.Client()).Execute(t.Context(), req); err == nil || calls.Load() != 0 {
				t.Fatalf("invalid payload was accepted: error = %v; calls = %d", err, calls.Load())
			}
		})
	}
}
