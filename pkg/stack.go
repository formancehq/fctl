package fctl

import (
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v3/collectionutils"
	"github.com/spf13/cobra"
)

func StackCompletion(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := LoadConfig(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	profile, _, err := LoadCurrentProfile(cmd, *cfg)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	organizationID, _ := ResolveOrganizationID(cmd, *profile)

	organizationClaims := profile.RootTokens.ID.Claims.Organizations
	organizationClaims = collectionutils.Filter(organizationClaims, func(s OrganizationAccess) bool {
		return strings.HasPrefix(s.ID, organizationID)
	})
	stackClaims := collectionutils.Map(organizationClaims, func(s OrganizationAccess) []StackAccess {
		return s.Stacks
	})
	stackList := collectionutils.Flatten(stackClaims)
	stackList = collectionutils.Filter(stackList, func(s StackAccess) bool {
		return toComplete == "" || strings.HasPrefix(s.ID, toComplete)
	})

	ret := collectionutils.Map(stackList, func(from StackAccess) string {
		return fmt.Sprintf("%s\t%s", from.ID, from.DisplayName)
	})

	return ret, cobra.ShellCompDirectiveNoFileComp
}
