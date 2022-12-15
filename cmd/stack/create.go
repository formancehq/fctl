package stack

import (
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	const productionFlag = "production"
	return fctl.NewMembershipCommand("create",
		fctl.WithShortDescription("Create a new stack"),
		fctl.WithAliases("c", "cr"),
		fctl.WithArgs(cobra.ExactArgs(1)),
		fctl.WithBoolFlag(productionFlag, false, ""),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}

			organization, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			apiClient, err := fctl.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			production := fctl.GetBool(cmd, productionFlag)
			stack, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
				Name:       args[0],
				Production: &production,
			}).Execute()
			if err != nil {
				return fctl.WrapError(err, "creating stack")
			}

			profile := fctl.GetCurrentProfile(cmd, cfg)

			if err := waitStackReady(cmd, profile, stack.Data); err != nil {
				return err
			}

			fctl.Highlightln(cmd.OutOrStdout(), "Your dashboard will be reachable on: %s",
				profile.ServicesBaseUrl(stack.Data).String())
			fctl.Highlightln(cmd.OutOrStdout(), "You can access your stack apis using following urls :")

			return internal.PrintStackInformation(cmd.OutOrStdout(), profile, stack.Data)
		}),
	)
}

func waitStackReady(cmd *cobra.Command, profile *fctl.Profile, stack *membershipclient.Stack) error {
	baseUrlStr := profile.ServicesBaseUrl(stack).String()
	authServerUrl := fmt.Sprintf("%s/api/auth", baseUrlStr)
	for {
		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet,
			fmt.Sprintf(authServerUrl+"/.well-known/openid-configuration"), nil)
		if err != nil {
			return err
		}
		rsp, err := fctl.GetHttpClient(cmd).Do(req)
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
