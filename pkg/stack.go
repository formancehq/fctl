package fctl

import (
	"fmt"

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
