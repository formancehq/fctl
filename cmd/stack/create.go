package stack

import (
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	return cmdbuilder.NewMembershipCommand("create",
		cmdbuilder.WithShortDescription("Create a new sandbox"),
		cmdbuilder.WithAliases("c", "cr"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}

			organization, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return err
			}

			apiClient, err := config.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			stack, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
				Name: args[0],
			}).Execute()
			if err != nil {
				return errors.Wrap(err, "creating sandbox")
			}

			profile := config.GetCurrentProfile(cmd, cfg)

			if err := waitStackReady(cmd, profile, stack.Data); err != nil {
				return err
			}

			cmdbuilder.Highlightln(cmd.OutOrStdout(), "Your dashboard will be reachable on: %s",
				profile.ServicesBaseUrl(stack.Data.OrganizationId, stack.Data.Id).String())
			cmdbuilder.Highlightln(cmd.OutOrStdout(), "You can access your sandbox apis using following urls :")

			return internal.PrintStackInformation(cmd.OutOrStdout(), profile, stack.Data)
		}),
	)
}

func waitStackReady(cmd *cobra.Command, profile *config.Profile, stack *membershipclient.Stack) error {
	baseUrlStr := profile.ServicesBaseUrl(stack.OrganizationId, stack.Id).String()
	authServerUrl := fmt.Sprintf("%s/api/auth", baseUrlStr)
	for {
		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet,
			fmt.Sprintf(authServerUrl+"/.well-known/openid-configuration"), nil)
		if err != nil {
			return err
		}
		rsp, err := config.GetHttpClient(cmd).Do(req)
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
