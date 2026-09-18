package fctl

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestResolveOrganizationID(t *testing.T) {
	for _, tc := range []struct {
		name      string
		profile   Profile
		flag, env string
		want      string
		wantErr   error
	}{
		{name: "disconnected", wantErr: ErrOrganizationNotSpecified},
		{name: "no organizations", profile: contextProfile(), wantErr: ErrOrganizationNotSpecified},
		{name: "unique organization", profile: contextProfile(OrganizationAccess{ID: "only"}), want: "only"},
		{name: "ambiguous organizations", profile: contextProfile(OrganizationAccess{ID: "first"}, OrganizationAccess{ID: "second"}), wantErr: ErrMultipleOrganizationsFound},
		{name: "saved default", profile: Profile{DefaultOrganization: "saved"}, want: "saved"},
		{name: "environment overrides saved default", profile: Profile{DefaultOrganization: "saved"}, env: "environment", want: "environment"},
		{name: "flag overrides environment and saved default", profile: Profile{DefaultOrganization: "saved"}, flag: "explicit", env: "environment", want: "explicit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ORGANIZATION", tc.env)
			cmd := &cobra.Command{}
			cmd.Flags().String(OrganizationFlag, "", "")
			if tc.flag != "" {
				require.NoError(t, cmd.ParseFlags([]string{"--organization", tc.flag}))
			}
			id, err := ResolveOrganizationID(cmd, tc.profile)
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.want, id)
		})
	}
}

func TestResolveStackID(t *testing.T) {
	one := OrganizationAccess{ID: "one", Stacks: []StackAccess{{ID: "only"}}}
	many := OrganizationAccess{ID: "many", Stacks: []StackAccess{{ID: "first"}, {ID: "second"}}}
	for _, tc := range []struct {
		name               string
		profile            Profile
		args               []string
		env                string
		wantOrg, wantStack string
		wantErr            string
	}{
		{name: "disconnected", wantErr: ErrOrganizationNotSpecified.Error()},
		{name: "no organizations", profile: contextProfile(), wantErr: ErrOrganizationNotSpecified.Error()},
		{name: "ambiguous organizations even with explicit stack", profile: contextProfile(one, many), args: []string{"--stack=second"}, wantErr: ErrMultipleOrganizationsFound.Error()},
		{name: "no stacks", profile: contextProfile(OrganizationAccess{ID: "empty"}), wantErr: ErrNoStackSpecified.Error()},
		{name: "unique stack", profile: contextProfile(one), wantOrg: "one", wantStack: "only"},
		{name: "ambiguous stacks", profile: contextProfile(many), wantErr: "found more than one stack and no stack specified"},
		{name: "selected organization scopes inference", profile: contextProfile(many, one), args: []string{"--organization=one"}, wantOrg: "one", wantStack: "only"},
		{name: "unknown organization cannot infer stack", profile: contextProfile(one), args: []string{"--organization=unknown"}, wantErr: ErrOrganizationNotSpecified.Error()},
		{name: "saved defaults", profile: Profile{DefaultOrganization: "saved-org", DefaultStack: "saved-stack"}, wantOrg: "saved-org", wantStack: "saved-stack"},
		{name: "environment overrides saved stack", profile: Profile{DefaultOrganization: "saved-org", DefaultStack: "saved-stack"}, env: "environment", wantOrg: "saved-org", wantStack: "environment"},
		{name: "flag overrides environment and saved stack", profile: Profile{DefaultOrganization: "saved-org", DefaultStack: "saved-stack"}, args: []string{"--stack=explicit"}, env: "environment", wantOrg: "saved-org", wantStack: "explicit"},
		{name: "explicit selection resolves ambiguity", profile: contextProfile(one, many), args: []string{"--organization=many", "--stack=second"}, wantOrg: "many", wantStack: "second"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ORGANIZATION", "")
			t.Setenv("STACK", tc.env)
			cmd := &cobra.Command{}
			cmd.Flags().String(OrganizationFlag, "", "")
			cmd.Flags().String(StackFlag, "", "")
			require.NoError(t, cmd.ParseFlags(tc.args))
			organization, stack, err := ResolveStackID(cmd, tc.profile)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantOrg, organization)
			require.Equal(t, tc.wantStack, stack)
		})
	}
}

func contextProfile(organizations ...OrganizationAccess) Profile {
	return Profile{RootTokens: &Tokens{ID: IDToken{Claims: IDTokenClaims{Organizations: organizations}}}}
}
