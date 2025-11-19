package fctl

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/formancehq/go-libs/v3/collectionutils"
)

func OrganizationCompletion(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := LoadConfig(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	profile, _, err := LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	organizationClaims := profile.RootTokens.ID.Claims.Organizations
	organizationClaims = collectionutils.Filter(organizationClaims, func(s OrganizationAccess) bool {
		return toComplete == "" || strings.HasPrefix(s.ID, toComplete)
	})

	ret := collectionutils.Map(organizationClaims, func(from OrganizationAccess) string {
		return fmt.Sprintf("%s\t%s", from.ID, from.DisplayName)
	})

	return ret, cobra.ShellCompDirectiveNoFileComp
}
