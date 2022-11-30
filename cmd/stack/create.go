package stack

import (
	"fmt"
	"net/http"
	"time"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	return internal2.NewMembershipCommand("create",
		internal2.WithShortDescription("Create a new sandbox"),
		internal2.WithAliases("c", "cr"),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			organization, err := internal2.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			apiClient, err := internal2.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			stack, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
				Name: args[0],
			}).Execute()
			if err != nil {
				return internal2.WrapError(err, "creating sandbox")
			}

			profile := internal2.GetCurrentProfile(cmd, cfg)

			if err := waitStackReady(cmd, profile, stack.Data); err != nil {
				return err
			}

			internal2.Highlightln(cmd.OutOrStdout(), "Your dashboard will be reachable on: %s",
				profile.ServicesBaseUrl(stack.Data).String())
			internal2.Highlightln(cmd.OutOrStdout(), "You can access your sandbox apis using following urls :")

			return internal.PrintStackInformation(cmd.OutOrStdout(), profile, stack.Data)
		}),
	)
}

func waitStackReady(cmd *cobra.Command, profile *internal2.Profile, stack *membershipclient.Stack) error {
	baseUrlStr := profile.ServicesBaseUrl(stack).String()
	authServerUrl := fmt.Sprintf("%s/api/auth", baseUrlStr)
	for {
		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet,
			fmt.Sprintf(authServerUrl+"/.well-known/openid-configuration"), nil)
		if err != nil {
			return err
		}
		rsp, err := internal2.GetHttpClient(cmd).Do(req)
		if err == nil && rsp.StatusCode == http.StatusOK {
			break
		}
		select {
		case <-cmd.Context().Done():
		case <-time.After(time.Second):
		}
	}
	return nil
}
