package integration_test

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type cliCloudOrganization struct {
	id, idToken, membership, refreshToken, deviceCode string
	devices, polls                                    atomic.Int32
}

func (f *cliCloudFixture) prepareOrganizations(t *testing.T, signer jose.Signer) {
	t.Helper()
	f.organizations = make(map[string]*cliCloudOrganization)
	for _, target := range f.targets {
		if f.organizations[target.organization] != nil {
			continue
		}
		org := &cliCloudOrganization{
			id: target.organization, refreshToken: "organization-refresh-" + target.organization,
			deviceCode: "organization-device-" + target.organization,
		}
		var targets []*cliCloudTarget
		for _, candidate := range f.targets {
			if candidate.organization == org.id {
				targets = append(targets, candidate)
			}
		}
		org.idToken = f.signIdentity(t, signer, targets, "organization/"+org.id)
		raw, err := jwt.Signed(signer).Claims(map[string]any{
			"iss": f.issuer(), "aud": "fctl", "sub": "cli-user", "jti": "organization-access/" + org.id,
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			"organization_id": org.id, "scope": "openid offline_access organization:ReadStack",
		}).Serialize()
		if err != nil {
			t.Fatal(err)
		}
		org.membership = raw
		f.organizations[org.id] = org
	}
}

func (f *cliCloudFixture) organizationDevice(t *testing.T, w http.ResponseWriter, r *http.Request, id string) (string, bool) {
	t.Helper()
	org := f.organizations[id]
	if org == nil {
		t.Error("device grant requested an unauthorized organization")
		w.WriteHeader(http.StatusForbidden)
		return "", false
	}
	org.devices.Add(1)
	assertCLICloudForm(t, r, map[string]string{
		"organization_id": id, "id_token_hint": f.idToken,
		"scope": "openid offline_access organization:ReadStack",
	})
	assertCLICloudAbsentForm(t, r, "resource", "prompt")
	return org.deviceCode, true
}

func (f *cliCloudFixture) organizationPoll(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	for _, org := range f.organizations {
		if r.PostForm.Get("device_code") != org.deviceCode {
			continue
		}
		org.polls.Add(1)
		assertCLICloudAbsentForm(t, r, "scope", "resource")
		writeCLICloudJSON(t, w, map[string]any{
			"access_token": org.membership, "refresh_token": org.refreshToken,
			"id_token": org.idToken, "token_type": "Bearer", "expires_in": 3600,
		})
		return true
	}
	return false
}

func (f *cliCloudFixture) organizationStack(t *testing.T, w http.ResponseWriter, r *http.Request, target *cliCloudTarget) {
	t.Helper()
	f.stackVersions.Add(1)
	org := f.organizations[target.organization]
	if r.Header.Get("Authorization") != "Bearer "+org.membership {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	writeCLICloudJSON(t, w, map[string]any{"data": map[string]string{
		"id": target.stack, "organizationId": org.id, "version": "v4.0",
	}})
}

func assertCLICloudAbsentForm(t *testing.T, r *http.Request, names ...string) {
	t.Helper()
	for _, name := range names {
		if r.PostForm.Has(name) {
			t.Errorf("unexpected %s form field at %s", name, r.URL.Path)
		}
	}
}

func (f *cliCloudFixture) assertOrganizationSaved(t *testing.T, dir, id string) {
	t.Helper()
	root := readCLICloudStore(t, dir).Connections["cloud"].Session
	if root == nil || root.Organizations[id] == nil {
		t.Fatal("organization grant was not persisted")
	}
	saved, org := root.Organizations[id], f.organizations[id]
	if saved.Options.Organization != id || saved.Options.Stack != "" || saved.Options.Issuer != f.issuer() || saved.Options.ClientID != "fctl" || saved.StackToken != nil || saved.StackURL != "" {
		t.Fatal("organization grant contains another identity or stack credentials")
	}
	if saved.IDToken != org.idToken || saved.MembershipToken == nil || saved.MembershipToken.AccessToken != org.membership || saved.MembershipToken.RefreshToken != org.refreshToken || !saved.MembershipToken.Expiry.After(time.Now()) {
		t.Fatal("saved organization credentials differ from the verified grant")
	}
	if org.devices.Load() != 1 || org.polls.Load() != 1 {
		t.Fatal("organization authorization was repeated instead of reusing its cached grant")
	}
}

func TestCloudCLIOrganizationGrantSharedAcrossStacks(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t, true)
	dir := t.TempDir()
	f.run(t, dir, "login", "--issuer", f.issuer())
	for _, stack := range []string{"stack", "second", "stack"} {
		out, _ := f.run(t, dir, "--organization", "org", "--stack", stack, "ledger", "list")
		assertCLICloudJSON(t, out, `{"data":[]}`)
		f.assertTargetSaved(t, dir, "org", stack)
	}
	f.assertCounts(t, 4, 4, 2, 3, 0, 3)
	f.assertRootSession(t, dir)
	root := readCLICloudStore(t, dir).Connections["cloud"].Session
	if len(root.Organizations) != 1 || len(root.Targets) != 2 || f.organizations["other"].devices.Load() != 0 {
		t.Fatal("stack selection authorized another organization or lost cached grants")
	}
}

func TestCloudStackVersionFixtureRequiresOrganizationBearer(t *testing.T) {
	t.Parallel()
	f := newCLICloudFixture(t, true)
	for _, token := range []string{"", cliMembershipToken, f.targets[0].membership, f.targets[0].stackToken, f.organizations["other"].membership, f.organizations["org"].membership} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.issuer()+"/organizations/org/stacks/stack", nil)
		if err != nil {
			t.Fatal(err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := f.server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		want := http.StatusUnauthorized
		if token == f.organizations["org"].membership {
			want = http.StatusOK
		}
		if response.StatusCode != want {
			t.Errorf("Membership stack route returned %d, want %d", response.StatusCode, want)
		}
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}
}
