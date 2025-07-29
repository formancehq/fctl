package fctl

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	"github.com/formancehq/go-libs/collectionutils"
	"github.com/spf13/cobra"
)

type MembershipStackStore struct {
	*OrganizationStore
	stackId string
}

func (cns MembershipStackStore) StackId() string {
	return cns.stackId
}

func NewMembershipStackStore(cmd *cobra.Command) error {
	cfg, err := GetConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	stackId, err := GetSelectedStackIDError(cmd, cfg)
	if err != nil {
		return err
	}

	store := GetOrganizationStore(cmd)
	cmd.SetContext(ContextWithMembershipStackStore(
		cmd.Context(),
		&MembershipStackStore{
			OrganizationStore: store,
			stackId:           stackId,
		},
	))

	return nil
}

func StackCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if err := NewMembershipStore(cmd); err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	var organization string
	if orgaFlag := cmd.Flag(organizationFlag); orgaFlag != nil {
		organization = orgaFlag.Value.String()
	}

	mbStore := GetMembershipStore(cmd.Context())
	if mbStore == nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	if organization == "" {
		if mbStore.Config == nil {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		p := mbStore.Config.GetProfile(GetCurrentProfileName(cmd, mbStore.Config))
		if p != nil {
			organization = p.GetDefaultOrganization()
		}
	}

	if organization == "" {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	ret, res, err := mbStore.Client().ListStacks(cmd.Context(), organization).Execute()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveError
	}

	if res.StatusCode > 300 {
		return []string{}, cobra.ShellCompDirectiveError
	}

	opts := collectionutils.Reduce(ret.Data, func(acc []string, s membershipclient.Stack) []string {
		return append(acc, fmt.Sprintf("%s\t%s", s.Id, s.Name))
	}, []string{})

	return opts, cobra.ShellCompDirectiveNoFileComp
}
