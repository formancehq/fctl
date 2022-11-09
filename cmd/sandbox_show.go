package cmd

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal"
	"github.com/numary/membership-api/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newSandboxShowCommand() *cobra.Command {
	const stackNameFlag = "stack"

	return newMembershipCommand("show",
		withShortDescription("show sandbox"),
		withArgs(cobra.MaximumNArgs(1)),
		withStringFlag(stackNameFlag, "", ""),
		withRunE(func(cmd *cobra.Command, args []string) error {
			organization, err := resolveOrganizationID(cmd)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := newMembershipClient(cmd)
			if err != nil {
				return err
			}

			var stack *client.Stack
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

			profile, err := getCurrentProfile()
			if err != nil {
				return err
			}

			return internal.PrintStackInformation(cmd.OutOrStdout(), profile, stack)
		}),
	)
}
