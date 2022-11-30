package stack

import (
	"net/http"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var errStackNotFound = errors.New("stack not found")

func NewShowCommand() *cobra.Command {
	const stackNameFlag = "name"

	return internal2.NewMembershipCommand("show",
		internal2.WithAliases("s", "sh"),
		internal2.WithShortDescription("Show sandbox"),
		internal2.WithArgs(cobra.MaximumNArgs(1)),
		internal2.WithStringFlag(stackNameFlag, "", ""),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}
			organization, err := internal2.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := internal2.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			var stack *membershipclient.Stack
			if len(args) == 1 {
				if internal2.GetString(cmd, stackNameFlag) != "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				stackResponse, httpResponse, err := apiClient.DefaultApi.ReadStack(cmd.Context(), organization, args[0]).Execute()
				if err != nil {
					if httpResponse.StatusCode == http.StatusNotFound {
						return errStackNotFound
					}
					return errors.Wrap(err, "listing stacks")
				}
				stack = stackResponse.Data
			} else {
				if internal2.GetString(cmd, stackNameFlag) == "" {
					return errors.New("need either an id of a name specified using --name flag")
				}
				stacksResponse, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				for _, s := range stacksResponse.Data {
					if s.Name == internal2.GetString(cmd, stackNameFlag) {
						stack = &s
						break
					}
				}
			}

			if stack == nil {
				return errStackNotFound
			}

			return internal.PrintStackInformation(cmd.OutOrStdout(), internal2.GetCurrentProfile(cmd, cfg), stack)
		}),
	)
}
