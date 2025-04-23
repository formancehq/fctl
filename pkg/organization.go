package fctl

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

type OrganizationStore struct {
	*MembershipStore
	organizationId string
}

func (cns OrganizationStore) Client() *membershipclient.DefaultAPIService {
	return cns.MembershipClient.DefaultAPI
}

func (cns OrganizationStore) OrganizationId() string {
	return cns.organizationId
}

func (cns *OrganizationStore) NewStackStore(os *OrganizationStore, stackId string) *MembershipStackStore {
	return &MembershipStackStore{
		OrganizationStore: os,
		stackId:           stackId,
	}
}

func (cns *OrganizationStore) CheckRegionCapability(key string, checker func([]any) bool) func(cmd *cobra.Command, args []string) (err error) {
	return func(cmd *cobra.Command, args []string) (err error) {
		stack, err := ResolveStack(cmd, cns.Config, cns.organizationId)
		if err != nil {
			return
		}

		region, _, err := cns.Client().GetRegion(cmd.Context(), cns.organizationId, stack.RegionID).Execute()
		if err != nil {
			return
		}

		capabilities, err := StructToMap(region.Data.Capabilities)
		if err != nil {
			return
		}

		if value, ok := capabilities[key]; ok {
			if values := value.([]interface{}); len(values) > 0 {
				if !checker(values) {
					return fmt.Errorf("unsupported membership server version: %s", value)
				}

			}
		}
		return
	}
}

func NewOrganizationStore(store *MembershipStore, organization string) *OrganizationStore {
	return &OrganizationStore{
		MembershipStore: store,
		organizationId:  organization,
	}
}

func NewMembershipOrganizationStore(cmd *cobra.Command) error {
	if err := NewMembershipStore(cmd); err != nil {
		return err
	}

	store := GetMembershipStore(cmd.Context())
	organization, err := ResolveOrganizationID(cmd, store.Config, store.Client())
	if err != nil {
		return err
	}

	cmd.SetContext(ContextWithOrganizationStore(cmd.Context(), NewOrganizationStore(store, organization)))

	return nil
}
