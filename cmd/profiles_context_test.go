package cmd_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/formancehq/go-libs/v4/oidc"

	"github.com/formancehq/fctl/v3/cmd"
	fctl "github.com/formancehq/fctl/v3/pkg"
)

func newProfileContext(t *testing.T) (*cobra.Command, string) {
	t.Helper()
	for _, env := range []string{"PROFILE", "ORGANIZATION", "STACK"} {
		t.Setenv(env, "")
	}
	dir := t.TempDir()
	storage := &cobra.Command{}
	storage.Flags().String(fctl.ConfigDir, dir, "")
	return storage, dir
}

func executeProfileContext(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	root := cmd.NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(append([]string{"--config-dir", dir, "--telemetry=false"}, args...))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := root.ExecuteContext(ctx)
	t.Logf("stdout: %s\nstderr: %s", stdout.String(), stderr.String())
	return stdout.String(), err
}

func TestProfilesOfflineLifecycle(t *testing.T) {
	storage, dir := newProfileContext(t)
	profile := fctl.Profile{MembershipURI: "https://membership.invalid", DefaultOrganization: "org", DefaultStack: "stack"}
	require.NoError(t, fctl.WriteProfile(storage, "work", profile))
	require.NoError(t, fctl.WriteProfile(storage, "other", fctl.Profile{}))

	out, err := executeProfileContext(t, dir, "profiles", "use", "work")
	require.NoError(t, err)
	require.Contains(t, out, "Selected profile updated!")
	cfg, err := fctl.LoadConfig(storage)
	require.NoError(t, err)
	require.Equal(t, "work", cfg.CurrentProfile)

	out, err = executeProfileContext(t, dir, "profiles", "list")
	require.NoError(t, err)
	require.Regexp(t, `work\s+\|\s+Yes`, out)
	require.Regexp(t, `other\s+\|\s+No`, out)
	out, err = executeProfileContext(t, dir, "--profile=other", "profiles", "list")
	require.NoError(t, err)
	require.Regexp(t, `other\s+\|\s+Yes`, out)
	require.Regexp(t, `work\s+\|\s+No`, out)
	cfg, err = fctl.LoadConfig(storage)
	require.NoError(t, err)
	require.Equal(t, "work", cfg.CurrentProfile, "--profile must not change the saved selection")

	out, err = executeProfileContext(t, dir, "profiles", "show", "work")
	require.NoError(t, err)
	require.Contains(t, out, profile.MembershipURI)
	require.Contains(t, out, profile.DefaultOrganization)
	require.Contains(t, out, profile.DefaultStack)

	out, err = executeProfileContext(t, dir, "profiles", "rename", "work", "renamed")
	require.NoError(t, err)
	require.Contains(t, out, "Profile renamed!")
	cfg, err = fctl.LoadConfig(storage)
	require.NoError(t, err)
	require.Equal(t, "renamed", cfg.CurrentProfile)
	_, err = fctl.LoadProfile(storage, "work")
	require.ErrorIs(t, err, os.ErrNotExist)
	loaded, err := fctl.LoadProfile(storage, "renamed")
	require.NoError(t, err)
	require.Equal(t, profile, *loaded)

	out, err = executeProfileContext(t, dir, "profiles", "reset", "renamed")
	require.NoError(t, err)
	require.Contains(t, out, "Profile reset")
	loaded, err = fctl.LoadProfile(storage, "renamed")
	require.NoError(t, err)
	profile.MembershipURI = fctl.DefaultMembershipURI
	require.Equal(t, profile, *loaded)

	out, err = executeProfileContext(t, dir, "profiles", "delete", "other")
	require.NoError(t, err)
	require.Contains(t, out, "Profile deleted!")
	names, err := fctl.ListProfiles(storage)
	require.NoError(t, err)
	require.Equal(t, []string{"renamed"}, names)
}

func TestProfilesUseRejectsMissingOrInvalidProfile(t *testing.T) {
	for _, name := range []string{"missing", "invalid"} {
		t.Run(name, func(t *testing.T) {
			storage, dir := newProfileContext(t)
			require.NoError(t, fctl.WriteConfig(storage, fctl.Config{CurrentProfile: "saved"}))
			if name == "invalid" {
				require.NoError(t, fctl.WriteProfile(storage, name, fctl.Profile{}))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "profiles", name, "profile.json"), []byte("{broken"), 0600))
			}
			_, err := executeProfileContext(t, dir, "profiles", "use", name)
			require.Error(t, err)
			cfg, err := fctl.LoadConfig(storage)
			require.NoError(t, err)
			require.Equal(t, "saved", cfg.CurrentProfile)
		})
	}
}

func TestProfilesSetDefaultsTargetsSelectedProfile(t *testing.T) {
	storage, dir := newProfileContext(t)
	profile := fctl.Profile{DefaultOrganization: "old", DefaultStack: "old-stack", RootTokens: &fctl.Tokens{
		ID: fctl.IDToken{Claims: fctl.IDTokenClaims{Organizations: []fctl.OrganizationAccess{
			{ID: "new", Stacks: []fctl.StackAccess{{ID: "new-stack"}}},
		}}},
	}}
	require.NoError(t, fctl.WriteProfile(storage, "saved", profile))
	require.NoError(t, fctl.WriteProfile(storage, "selected", profile))
	require.NoError(t, fctl.WriteConfig(storage, fctl.Config{CurrentProfile: "saved"}))
	t.Setenv("PROFILE", "saved")

	out, err := executeProfileContext(t, dir, "--profile=selected", "profiles", "set-default-organization", "new")
	require.NoError(t, err)
	require.Contains(t, out, "Default organization updated!")
	selected, err := fctl.LoadProfile(storage, "selected")
	require.NoError(t, err)
	require.Equal(t, "new", selected.DefaultOrganization)
	require.Empty(t, selected.DefaultStack, "switching organization clears the previous stack")

	out, err = executeProfileContext(t, dir, "--profile=selected", "profiles", "set-default-stack", "new-stack")
	require.NoError(t, err)
	require.Contains(t, out, "Default stack updated!")
	selected, err = fctl.LoadProfile(storage, "selected")
	require.NoError(t, err)
	require.Equal(t, "new-stack", selected.DefaultStack)
	_, err = executeProfileContext(t, dir, "--profile=selected", "profiles", "set-default-stack", "unknown")
	require.ErrorContains(t, err, "not found in your access list")
	unchanged, err := fctl.LoadProfile(storage, "selected")
	require.NoError(t, err)
	require.Equal(t, selected, unchanged)
	saved, err := fctl.LoadProfile(storage, "saved")
	require.NoError(t, err)
	require.Equal(t, profile, *saved)
}

func TestProfilesSetDefaultStackDisconnected(t *testing.T) {
	for _, organization := range []string{"", "saved-org"} {
		t.Run("organization="+organization, func(t *testing.T) {
			storage, dir := newProfileContext(t)
			profile := fctl.Profile{DefaultOrganization: organization}
			require.NoError(t, fctl.WriteProfile(storage, "default", profile))
			_, err := executeProfileContext(t, dir, "profiles", "set-default-stack", "stack")
			require.ErrorContains(t, err, "please run 'fctl login'")
			unchanged, err := fctl.LoadProfile(storage, "default")
			require.NoError(t, err)
			require.Equal(t, profile, *unchanged)
		})
	}
}

func TestRootContextSelection(t *testing.T) {
	for _, tc := range []struct {
		name, envProfile, envOrg, envStack string
		args                               []string
		wantErr                            string
	}{
		{name: "explicit flags override environment and saved defaults", envProfile: "missing", envOrg: "wrong", envStack: "wrong", args: []string{"--profile=selected", "--organization=org-b", "--stack=stack-b"}},
		{name: "environment overrides saved defaults", envProfile: "selected", envOrg: "org-b", envStack: "stack-b"},
		{name: "saved profile defaults"},
		{name: "ambiguous organizations", args: []string{"--profile=ambiguous"}, wantErr: fctl.ErrMultipleOrganizationsFound.Error()},
		{name: "ambiguous stacks", args: []string{"--profile=ambiguous", "--organization=org-b"}, wantErr: "found more than one stack and no stack specified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			storage, dir := newProfileContext(t)
			requests := make(chan string, 4)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case requests <- r.Method + " " + r.URL.Path + " " + r.Header.Get("Authorization"):
				default:
					t.Error("unexpected extra API request")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[]}`))
			}))
			defer server.Close()
			profile := fctl.Profile{MembershipURI: server.URL, DefaultOrganization: "org-b", DefaultStack: "stack-b", RootTokens: &fctl.Tokens{
				ID: fctl.IDToken{Claims: fctl.IDTokenClaims{Organizations: []fctl.OrganizationAccess{
					{ID: "org-a"}, {ID: "org-b", Stacks: []fctl.StackAccess{{ID: "stack-a"}, {ID: "stack-b"}}},
				}}},
			}}
			require.NoError(t, fctl.WriteProfile(storage, "saved", profile))
			require.NoError(t, fctl.WriteConfig(storage, fctl.Config{CurrentProfile: "saved"}))
			profile.DefaultOrganization, profile.DefaultStack = "org-a", "stack-a"
			require.NoError(t, fctl.WriteProfile(storage, "selected", profile))
			profile.DefaultOrganization, profile.DefaultStack = "", ""
			require.NoError(t, fctl.WriteProfile(storage, "ambiguous", profile))
			for _, name := range []string{"saved", "selected"} {
				token := fctl.AccessToken{}
				token.Token = name + "-token"
				token.Claims.OrganizationID = "org-b"
				token.Claims.Expiration = oidc.Time(time.Now().Add(time.Hour).Unix())
				require.NoError(t, fctl.WriteOrganizationToken(storage, name, token))
			}
			t.Setenv("PROFILE", tc.envProfile)
			t.Setenv("ORGANIZATION", tc.envOrg)
			t.Setenv("STACK", tc.envStack)
			out, err := executeProfileContext(t, dir, append([]string{"stack", "modules", "list"}, tc.args...)...)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				require.Empty(t, requests, "ambiguous contexts must not reach the API")
				return
			}
			require.NoError(t, err)
			require.Contains(t, out, "Name")
			tokenProfile := "selected"
			if tc.envProfile == "" {
				tokenProfile = "saved"
			}
			require.Len(t, requests, 1)
			require.Equal(t, "GET /organizations/org-b/stacks/stack-b/modules Bearer "+tokenProfile+"-token", <-requests)
		})
	}
}
