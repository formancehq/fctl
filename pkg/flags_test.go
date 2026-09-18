package fctl

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetStringContextFlags(t *testing.T) {
	for _, flag := range []struct {
		name string
		env  string
	}{
		{name: ProfileFlag, env: "PROFILE"},
		{name: OrganizationFlag, env: "ORGANIZATION"},
		{name: StackFlag, env: "STACK"},
		{name: ConfigDir, env: "CONFIG_DIR"},
	} {
		t.Run(flag.name, func(t *testing.T) {
			for _, tc := range []struct {
				name         string
				missing      bool
				defaultValue string
				set          bool
				value        string
				env          string
				want         string
			}{
				{name: "explicit flag wins", set: true, value: "flag-value", env: "env-value", want: "flag-value"},
				{name: "unset flag uses environment", env: "env-value", want: "env-value"},
				{name: "explicit empty flag uses environment", set: true, env: "env-value", want: "env-value"},
				{name: "unregistered flag uses environment", missing: true, env: "env-value", want: "env-value"},
				{name: "nonempty registered default wins", defaultValue: "default-value", env: "env-value", want: "default-value"},
				{name: "no value", want: ""},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Setenv(flag.env, tc.env)
					cmd := &cobra.Command{}
					if !tc.missing {
						cmd.Flags().String(flag.name, tc.defaultValue, "")
					}
					if tc.set {
						require.NoError(t, cmd.Flags().Set(flag.name, tc.value))
					}

					require.Equal(t, tc.want, GetString(cmd, flag.name))
				})
			}
		})
	}
}
