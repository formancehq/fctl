package stack

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	internal2 "github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/formancehq/fctl/cmd/stack/internal"
	"github.com/formancehq/fctl/membershipclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newSandboxShowCommand() *cobra.Command {
	const stackNameFlag = "stack"

	return cmdbuilder.NewMembershipCommand("show",
		cmdbuilder.WithShortDescription("show sandbox"),
		cmdbuilder.WithArgs(cobra.MaximumNArgs(1)),
		cmdbuilder.WithStringFlag(stackNameFlag, "", ""),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			config, err := internal2.Get()
			if err != nil {
				return err
			}
			organization, err := cmdbuilder.ResolveOrganizationID(cmd, config)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := membership.NewClient(cmd.Context(), config)
			if err != nil {
				return err
			}

			var stack *membershipclient.Stack
			if len(args) == 1 {
				if viper.GetString(stackNameFlag) == "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				stackResponse, _, err := apiClient.DefaultApi.ReadStack(cmd.Context(), organization, args[0]).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				stack = stackResponse.Data
			} else {
				if viper.GetString(stackNameFlag) == "" {
					return errors.New("need either an id of a name specified using --name flag")
				}
				stacksResponse, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				for _, s := range stacksResponse.Data {
					if s.Name == viper.GetString(stackNameFlag) {
						stack = &s
						break
					}
				}
			}

			if stack == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "Not found.")
				return nil
			}

			profile, err := internal2.GetCurrentProfile(config)
			if err != nil {
				return err
			}

			return internal.PrintStackInformation(cmd.OutOrStdout(), profile, stack)
		}),
	)
}
