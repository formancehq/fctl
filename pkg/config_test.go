package fctl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentProfileName(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flag  string
		env   string
		saved string
		want  string
	}{
		{name: "flag overrides environment and saved profile", flag: "flag-profile", env: "env-profile", saved: "saved-profile", want: "flag-profile"},
		{name: "environment overrides saved profile", env: "env-profile", saved: "saved-profile", want: "env-profile"},
		{name: "saved profile", saved: "saved-profile", want: "saved-profile"},
		{name: "default profile", want: "default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PROFILE", tc.env)
			cmd := &cobra.Command{}
			cmd.Flags().String(ProfileFlag, "", "")
			if tc.flag != "" {
				require.NoError(t, cmd.Flags().Set(ProfileFlag, tc.flag))
			}

			require.Equal(t, tc.want, GetCurrentProfileName(cmd, Config{CurrentProfile: tc.saved}))
		})
	}
}

func TestLoadConfigMissing(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}
	cmd.Flags().String(ConfigDir, dir, "")

	config, err := LoadConfig(cmd)
	require.NoError(t, err)
	require.Equal(t, &Config{CurrentProfile: "default"}, config)
	require.NoFileExists(t, filepath.Join(dir, "config.yml"))
}

func TestLoadConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}
	cmd.Flags().String(ConfigDir, dir, "")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yml"), []byte(`{"currentProfile":`), 0600))

	config, err := LoadConfig(cmd)
	require.Error(t, err)
	require.Nil(t, config)
}

func TestWriteConfigRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "config")
	cmd := &cobra.Command{}
	cmd.Flags().String(ConfigDir, dir, "")

	want := Config{CurrentProfile: "selected-profile", UniqueID: "synthetic-id"}
	require.NoError(t, WriteConfig(cmd, want))
	require.FileExists(t, filepath.Join(dir, "config.yml"))
	config, err := LoadConfig(cmd)
	require.NoError(t, err)
	require.Equal(t, &want, config)

	// Replacing a longer configuration must not leave stale file contents.
	want = Config{CurrentProfile: "other"}
	require.NoError(t, WriteConfig(cmd, want))
	config, err = LoadConfig(cmd)
	require.NoError(t, err)
	require.Equal(t, &want, config)
	contents, err := os.ReadFile(filepath.Clean(filepath.Join(dir, "config.yml")))
	require.NoError(t, err)
	require.JSONEq(t, `{"currentProfile":"other"}`, string(contents))
}

func TestLoadCurrentProfile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		wantURI string
		wantErr bool
	}{
		{name: "missing profile uses default endpoint", wantURI: DefaultMembershipURI},
		{name: "saved profile", content: `{"membershipURI":"https://membership.invalid"}`, wantURI: "https://membership.invalid"},
		{name: "invalid profile does not fall back", content: `{broken`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PROFILE", "")
			dir := t.TempDir()
			cmd := &cobra.Command{}
			cmd.Flags().String(ConfigDir, dir, "")
			cmd.Flags().String(ProfileFlag, "selected", "")
			if tc.content != "" {
				profileDir := filepath.Join(dir, "profiles", "selected")
				require.NoError(t, os.MkdirAll(profileDir, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(profileDir, "profile.json"), []byte(tc.content), 0600))
			}
			profile, name, err := LoadCurrentProfile(cmd, Config{CurrentProfile: "other"})
			require.Equal(t, "selected", name)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantURI, profile.MembershipURI)
			require.False(t, profile.IsConnected())
		})
	}
}
