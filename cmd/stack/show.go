package stack

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	config "github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	const stackNameFlag = "name"

	return cmdbuilder.NewMembershipCommand("show",
		cmdbuilder.WithAliases("s", "sh"),
		cmdbuilder.WithShortDescription("Show sandbox"),
		cmdbuilder.WithArgs(cobra.MaximumNArgs(1)),
		cmdbuilder.WithStringFlag(stackNameFlag, "", ""),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}
			organization, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := config.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			var stack *membershipclient.Stack
			if len(args) == 1 {
				if cmdutils.GetString(cmd, stackNameFlag) != "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				stackResponse, _, err := apiClient.DefaultApi.ReadStack(cmd.Context(), organization, args[0]).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				stack = stackResponse.Data
			} else {
				if cmdutils.GetString(cmd, stackNameFlag) == "" {
					return errors.New("need either an id of a name specified using --name flag")
				}
				stacksResponse, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				for _, s := range stacksResponse.Data {
					if s.Name == cmdutils.GetString(cmd, stackNameFlag) {
						stack = &s
						break
					}
				}
			}

			if stack == nil {
				return errors.New("Not found.")
			}

			return internal.PrintStackInformation(cmd.OutOrStdout(), config.GetCurrentProfile(cmd, cfg), stack)
		}),
	)
}
