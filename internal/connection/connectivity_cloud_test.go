package connection

import (
	"io"
	"net/http"
	"testing"

	"github.com/formancehq/fctl/v4/internal/cloud"
)

//nolint:gocognit // Verifies a signed cached grant, route resolution and use of the scoped Stack token rather than root identity.
func TestConnectivityCloudUsesScopedStackClient(t *testing.T) {
	t.Parallel()
	f := newSelectionFixture(t)
	options := cloud.Options{Issuer: f.entry.Options.Issuer, ClientID: "fctl", Organization: "first", Stack: "one"}
	f.entry.Session.Targets = map[string]*cloud.Session{"first/one": {
		Options: options, IDToken: f.entry.Session.IDToken, StackURL: f.server.URL + "/one",
		MembershipToken: selectionToken("scoped-membership"), StackToken: selectionToken("connectivity-stack"),
	}}
	if err := Save(f.dir, Store{Active: "work", Connections: map[string]Entry{"work": f.entry}}); err != nil {
		t.Fatal(err)
	}
	previous := f.server.Config.Handler
	reads := 0
	f.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/one/api/connectivity/_info" {
			reads++
			if r.Header.Get("Authorization") != "Bearer connectivity-stack" {
				t.Error("Connectivity received the root Membership token")
			}
			if _, err := io.WriteString(w, `{"version":"fixture"}`); err != nil {
				t.Error(err)
			}
			return
		}
		previous.ServeHTTP(w, r)
	})
	s, cmd := f.settings(t)
	for flag, value := range map[string]string{"organization": "first", "stack": "one"} {
		if err := cmd.Root().PersistentFlags().Set(flag, value); err != nil {
			t.Fatal(err)
		}
	}
	client, err := s.Client(t.Context(), cmd, "connectivity")
	if err != nil {
		t.Fatal(err)
	}
	if client.Endpoint() != f.server.URL+"/one/api/connectivity" {
		t.Fatalf("Cloud endpoint=%s", client.Endpoint())
	}
	if _, err := client.Do(t.Context(), http.MethodGet, "/_info", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if reads != 1 || f.posts.Load() != 0 {
		t.Fatalf("reads=%d extra grant/exchange requests=%d", reads, f.posts.Load())
	}
}
