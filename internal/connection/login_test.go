package connection_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/cloud"
	"github.com/formancehq/fctl/v4/internal/connection"
)

func loginSettings(t *testing.T, dir string, args ...string) (*connection.Settings, *cobra.Command) {
	t.Helper()
	s := &connection.Settings{}
	args = append([]string{"--config-dir", dir}, args...)
	root := commandWith(s, args...)
	if err := root.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	return s, root
}

func isolateLoginEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CONNECTION", "CONFIG_DIR", "STACK_URL", "LEDGER_URL", "AUTH_URL", "AUTH_MODE", "TOKEN_URL", "CLIENT_ID", "SCOPES", "ISSUER", "ORGANIZATION", "STACK"} {
		t.Setenv("FCTL_"+key, "")
		// LookupEnv distinguishes absent settings from explicit empty overrides.
		// Setenv above registers restoration of the original environment.
		if err := os.Unsetenv("FCTL_" + key); err != nil {
			t.Fatal(err)
		}
	}
}

func saveLoginStore(t *testing.T, dir string, store connection.Store) {
	t.Helper()
	if err := connection.Save(dir, store); err != nil {
		t.Fatal(err)
	}
}

func loadLoginStore(t *testing.T, dir string) connection.Store {
	t.Helper()
	store, err := connection.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func loginStoreBytes(t *testing.T, dir string) []byte {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	data, err := root.ReadFile("connections.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLoginProfilePreparesCloudWithoutCreatingProfile(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"cloud", nil},
		{"new-cloud", []string{"--profile", "new-cloud"}},
		{"short-cloud", []string{"-p", "short-cloud"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateLoginEnvironment(t)
			dir := t.TempDir()
			s, root := loginSettings(t, dir, tc.args...)
			options, entry, gotName, gotDir, err := s.LoginProfile(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			want := connection.Options{AuthMode: "cloud", Issuer: cloud.DefaultIssuer, ClientID: "fctl"}
			if options != want || !reflect.DeepEqual(entry, connection.Entry{}) || gotName != tc.name || gotDir != dir {
				t.Fatalf("prepared options=%+v entry=%+v name=%q dir=%q", options, entry, gotName, gotDir)
			}
			assertLoginDirectoryEmpty(t, dir)
		})
	}
}

func assertLoginDirectoryEmpty(t *testing.T, dir string) {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := root.Stat("connections.json"); !os.IsNotExist(err) {
		t.Fatalf("preparation created a configuration file: %v", err)
	}
	if _, err := root.Stat("connections.lock"); !os.IsNotExist(err) {
		t.Fatalf("read-only preparation created a lock file: %v", err)
	}
}

func TestLoginProfileReusesActiveCloud(t *testing.T) {
	isolateLoginEnvironment(t)
	dir := t.TempDir()
	entry := connection.NewEntry(connection.Options{AuthMode: "cloud", Issuer: "https://saved.example", ClientID: "saved-client", Organization: "org", Stack: "stack"})
	entry.Session = &cloud.Session{IDToken: "existing-session"}
	before := connection.Store{Active: "work", Connections: map[string]connection.Entry{"work": entry}}
	saveLoginStore(t, dir, before)
	s, root := loginSettings(t, dir)
	options, prepared, name, _, err := s.LoginProfile(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if name != "work" || options != entry.Options || !reflect.DeepEqual(prepared, entry) {
		t.Fatalf("active Cloud profile was replaced: name=%q entry=%+v", name, prepared)
	}
	if after := loadLoginStore(t, dir); !reflect.DeepEqual(after, before) {
		t.Fatalf("unchanged identity mutated store: %+v", after)
	}
}

func TestLoginProfileImplicitLocalFallsBackUntilSessionSaved(t *testing.T) {
	isolateLoginEnvironment(t)
	dir := t.TempDir()
	local := connection.NewEntry(connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9000"})
	saveLoginStore(t, dir, connection.Store{Active: "local", Connections: map[string]connection.Entry{"local": local}})
	s, root := loginSettings(t, dir)
	options, entry, name, _, err := s.LoginProfile(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	before := loadLoginStore(t, dir)
	if name != "cloud" || before.Active != "local" || !reflect.DeepEqual(before.Connections["local"], local) {
		t.Fatalf("login preparation changed local profile: %+v", before)
	}
	if len(before.Connections) != 1 || entry.Revision != "" {
		t.Fatal("preparation created the fallback Cloud profile")
	}
	expected := entry.Revision
	session := &cloud.Session{Options: cloud.Options{Issuer: cloud.DefaultIssuer, ClientID: "fctl"}, IDToken: "new-session"}
	if err := connection.SaveLoginSession(t.Context(), dir, name, &expected, options, session); err != nil {
		t.Fatal(err)
	}
	after := loadLoginStore(t, dir)
	if after.Active != "cloud" || after.Connections["cloud"].Options != options || !reflect.DeepEqual(after.Connections["cloud"].Session, session) || !reflect.DeepEqual(after.Connections["local"], local) {
		t.Fatalf("successful login did not persist session and selection together: %+v", after)
	}
	if expected == entry.Revision || after.Connections["cloud"].Revision != expected {
		t.Fatal("successful login did not advance the caller's revision")
	}
	assertStaleLoginPreservesStore(t, dir, name, entry.Revision, options)
}

func TestLoginProfileRejectsLocalAndNonCloudModesWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		env  string
	}{
		{"explicit local", []string{"--profile", "local"}, ""},
		{"environment local", nil, "local"},
		{"none", []string{"--profile", "new", "--auth-mode", "none"}, ""},
		{"client credentials", []string{"--profile", "new", "--auth-mode", "client-credentials"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateLoginEnvironment(t)
			t.Setenv("FCTL_PROFILE", tc.env)
			dir := t.TempDir()
			local := connection.NewEntry(connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9000"})
			saveLoginStore(t, dir, connection.Store{Active: "local", Connections: map[string]connection.Entry{"local": local}})
			before := string(loginStoreBytes(t, dir))
			s, root := loginSettings(t, dir, tc.args...)
			if _, _, _, _, err := s.LoginProfile(t.Context(), root); err == nil {
				t.Fatal("accepted non-Cloud login")
			}
			if string(loginStoreBytes(t, dir)) != before {
				t.Fatal("rejected login changed persisted settings")
			}
		})
	}
}

func TestLoginProfileIdentityOverridePreservesTarget(t *testing.T) {
	for _, tc := range []struct {
		flag, value, issuer, client string
	}{
		{"--issuer", "https://new.example", "https://new.example", "old-client"},
		{"--client-id", "new-client", "https://old.example", "new-client"},
	} {
		t.Run(tc.flag, func(t *testing.T) {
			isolateLoginEnvironment(t)
			dir := t.TempDir()
			entry := connection.NewEntry(connection.Options{AuthMode: "cloud", Issuer: "https://old.example", ClientID: "old-client", Organization: "org", Stack: "stack"})
			entry.Session = &cloud.Session{IDToken: "old-session"}
			saveLoginStore(t, dir, connection.Store{Active: "work", Connections: map[string]connection.Entry{"work": entry}})
			before := string(loginStoreBytes(t, dir))
			want := entry.Options
			want.Issuer, want.ClientID = tc.issuer, tc.client
			s, root := loginSettings(t, dir, tc.flag, tc.value)
			options, prepared, _, _, err := s.LoginProfile(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			if options != want || !reflect.DeepEqual(prepared, entry) {
				t.Fatalf("identity preparation changed original entry or lost target: %+v", prepared)
			}
			if string(loginStoreBytes(t, dir)) != before {
				t.Fatal("identity preparation changed persisted session, options or revision")
			}
			assertSuccessfulLoginCommit(t, dir, "work", entry.Revision, options)
		})
	}
}

func TestLoginProfileFlagsOverrideEnvironment(t *testing.T) {
	for _, flags := range []bool{false, true} {
		t.Run(map[bool]string{false: "environment", true: "flags"}[flags], func(t *testing.T) {
			isolateLoginEnvironment(t)
			t.Setenv("FCTL_PROFILE", "environment")
			t.Setenv("FCTL_ISSUER", "https://environment.example")
			t.Setenv("FCTL_CLIENT_ID", "environment-client")
			t.Setenv("FCTL_AUTH_MODE", "cloud")
			wantName, wantIssuer, wantClient := "environment", "https://environment.example", "environment-client"
			var args []string
			if flags {
				t.Setenv("FCTL_AUTH_MODE", "none")
				args = []string{"--profile", "flag", "--issuer", "https://flag.example", "--client-id", "flag-client", "--auth-mode", "cloud"}
				wantName, wantIssuer, wantClient = "flag", "https://flag.example", "flag-client"
			}
			s, root := loginSettings(t, t.TempDir(), args...)
			options, _, name, _, err := s.LoginProfile(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			if name != wantName || options.Issuer != wantIssuer || options.ClientID != wantClient || options.AuthMode != "cloud" {
				t.Fatalf("wrong flag/environment precedence: name=%q options=%+v", name, options)
			}
		})
	}
}

func TestLoginProfilePreservesIdentityWhenTargetOrDefaultsChange(t *testing.T) {
	for _, tc := range []struct {
		name  string
		saved connection.Options
		args  []string
		want  connection.Options
	}{
		{
			name:  "default target",
			saved: connection.Options{AuthMode: "cloud", Issuer: cloud.DefaultIssuer, ClientID: "fctl", Organization: "old-org", Stack: "old-stack"},
			args:  []string{"--organization", "new-org", "--stack", "new-stack"},
			want:  connection.Options{AuthMode: "cloud", Issuer: cloud.DefaultIssuer, ClientID: "fctl", Organization: "new-org", Stack: "new-stack"},
		},
		{
			name:  "implicit identity defaults",
			saved: connection.Options{AuthMode: "cloud", Organization: "org", Stack: "stack"},
			want:  connection.Options{AuthMode: "cloud", Issuer: cloud.DefaultIssuer, ClientID: "fctl", Organization: "org", Stack: "stack"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateLoginEnvironment(t)
			dir := t.TempDir()
			entry := connection.NewEntry(tc.saved)
			entry.Session = &cloud.Session{IDToken: "existing-identity", Targets: map[string]*cloud.Session{"old-org/old-stack": {IDToken: "cached-target"}}}
			saveLoginStore(t, dir, connection.Store{Active: "work", Connections: map[string]connection.Entry{"work": entry}})
			before := string(loginStoreBytes(t, dir))
			s, root := loginSettings(t, dir, tc.args...)
			options, prepared, name, _, err := s.LoginProfile(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			if options != tc.want || !reflect.DeepEqual(prepared, entry) {
				t.Fatalf("target/default preparation changed original entry: %+v", prepared)
			}
			if string(loginStoreBytes(t, dir)) != before {
				t.Fatal("target/default preparation changed persisted settings or session")
			}
			assertSuccessfulLoginCommit(t, dir, name, entry.Revision, options)
		})
	}
}

func assertSuccessfulLoginCommit(t *testing.T, dir, name, revision string, options connection.Options) {
	t.Helper()
	expected := revision
	session := &cloud.Session{IDToken: "successfully-authenticated"}
	if err := connection.SaveLoginSession(t.Context(), dir, name, &expected, options, session); err != nil {
		t.Fatal(err)
	}
	store := loadLoginStore(t, dir)
	entry := store.Connections[name]
	if store.Active != name || entry.Options != options || !reflect.DeepEqual(entry.Session, session) {
		t.Fatal("successful login did not commit options, session and selection together")
	}
	if expected == "" || expected == revision || entry.Revision != expected {
		t.Fatal("successful login did not advance the persisted and caller revisions")
	}
	assertStaleLoginPreservesStore(t, dir, name, revision, options)
}

func assertStaleLoginPreservesStore(t *testing.T, dir, name, revision string, options connection.Options) {
	t.Helper()
	before := loadLoginStore(t, dir)
	expected := revision
	if err := connection.SaveLoginSession(t.Context(), dir, name, &expected, options, &cloud.Session{IDToken: "stale-login"}); err == nil {
		t.Fatal("settings change accepted login started with previous revision")
	}
	if after := loadLoginStore(t, dir); expected != revision || !reflect.DeepEqual(after, before) {
		t.Fatal("failed reauthentication discarded existing identity or changed settings")
	}
}

func TestSaveLoginSessionRejectsLogoutDeleteAndReplacement(t *testing.T) {
	for _, operation := range []string{"logout", "delete", "replace"} {
		t.Run(operation, func(t *testing.T) {
			isolateLoginEnvironment(t)
			dir := t.TempDir()
			local := connection.NewEntry(connection.Options{AuthMode: "none", LedgerURL: "http://localhost:9000"})
			cloudEntry := connection.NewEntry(connection.Options{AuthMode: "cloud", Issuer: cloud.DefaultIssuer, ClientID: "fctl"})
			cloudEntry.Session = &cloud.Session{IDToken: "existing-identity"}
			saveLoginStore(t, dir, connection.Store{Active: "local", Connections: map[string]connection.Entry{"local": local, "cloud": cloudEntry}})
			s, root := loginSettings(t, dir)
			options, entry, name, _, err := s.LoginProfile(t.Context(), root)
			if err != nil {
				t.Fatal(err)
			}
			expected := entry.Revision
			invalidateLoginEntry(t, dir, name, entry, operation)
			before := string(loginStoreBytes(t, dir))
			if err := connection.SaveLoginSession(t.Context(), dir, name, &expected, options, &cloud.Session{IDToken: "stale-session"}); err == nil {
				t.Fatal("accepted stale login result")
			}
			if expected != entry.Revision || string(loginStoreBytes(t, dir)) != before {
				t.Fatal("failed CAS mutated revision, session or active profile")
			}
		})
	}
}

func invalidateLoginEntry(t *testing.T, dir, name string, entry connection.Entry, operation string) {
	t.Helper()
	if operation == "logout" {
		expected := entry.Revision
		if err := connection.SaveSession(t.Context(), dir, name, &expected, nil); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := connection.Update(t.Context(), dir, func(store *connection.Store) error {
		if operation == "delete" {
			delete(store.Connections, name)
		} else {
			store.Connections[name] = connection.NewEntry(entry.Options)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSaveLoginSessionRejectsConcurrentCreationOfAbsentProfile(t *testing.T) {
	isolateLoginEnvironment(t)
	dir := t.TempDir()
	s, root := loginSettings(t, dir, "--profile", "new-cloud")
	options, entry, name, _, err := s.LoginProfile(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Revision != "" {
		t.Fatal("absent profile returned an existing revision")
	}
	assertLoginDirectoryEmpty(t, dir)
	concurrent := connection.NewEntry(options)
	concurrent.Session = &cloud.Session{IDToken: "other-command-identity"}
	saveLoginStore(t, dir, connection.Store{Active: name, Connections: map[string]connection.Entry{name: concurrent}})
	assertStaleLoginPreservesStore(t, dir, name, entry.Revision, options)
}

func TestValidateCloudAllowsIdentityWithoutTarget(t *testing.T) {
	for _, options := range []connection.Options{
		{AuthMode: "cloud"},
		{AuthMode: "cloud", Issuer: "https://issuer.example"},
		{AuthMode: "cloud", Organization: "org"},
		{AuthMode: "cloud", Stack: "stack"},
		{AuthMode: "cloud", Organization: "org", Stack: "stack"},
	} {
		if err := connection.Validate(options); err != nil {
			t.Errorf("Cloud identity with optional issuer/target rejected: %+v: %v", options, err)
		}
	}
}

func TestNoBrowserIsGlobalLoginSetting(t *testing.T) {
	isolateLoginEnvironment(t)
	s, root := loginSettings(t, t.TempDir())
	if s.NoBrowser || s.BrowserOpener() == nil {
		t.Fatal("default browser opening is disabled")
	}
	child := &cobra.Command{Use: "login"}
	root.AddCommand(child)
	if err := child.ParseFlags([]string{"--no-browser"}); err != nil {
		t.Fatal(err)
	}
	if !s.NoBrowser || s.BrowserOpener() != nil || !root.PersistentFlags().Changed("no-browser") {
		t.Fatal("login does not inherit global --no-browser setting")
	}
}
