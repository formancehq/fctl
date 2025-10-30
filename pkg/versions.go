package fctl

import (
	"fmt"

	"github.com/formancehq/fctl/membershipclient"
	"github.com/formancehq/go-libs/collectionutils"
	"github.com/spf13/cobra"
)

func CheckMembershipCapabilities(apiClient *membershipclient.APIClient, capability membershipclient.Capability) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		serverInfo, err := MembershipServerInfo(cmd.Context(), apiClient.DefaultAPI)
		if err != nil {
			return err
		}

		if collectionutils.Contains(serverInfo.Capabilities, capability) {
			return nil
		}

		return fmt.Errorf("unsupported membership server capability: %s", capability)
	}
}
