package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newSandboxDeleteCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)

	return newMembershipCommand("delete [STACK_ID] | --name=[NAME]",
		withShortDescription("delete a sandbox"),
		withArgs(cobra.MaximumNArgs(1)),
		withStringFlag(stackNameFlag, "", "Sandbox to remove"),
		withPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		}),
		withRunE(func(cmd *cobra.Command, args []string) error {
			organization, err := resolveOrganizationID(cmd)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := newMembershipClient(cmd)
			if err != nil {
				return err
			}

			var stackID string
			if len(args) == 1 {
				if viper.GetString(stackNameFlag) == "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				stackID = args[0]
			} else {
				if viper.GetString(stackNameFlag) == "" {
					return errors.New("need either an id of a name specified using --name flag")
				}
				stacks, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				for _, s := range stacks.Data {
					if s.Name == viper.GetString(stackNameFlag) {
						stackID = s.Id
						break
					}
				}
				if stackID == "" {
					return errors.New("sandbox not found")
				}
			}

			if _, err := apiClient.DefaultApi.DeleteStack(cmd.Context(), organization, stackID).Execute(); err != nil {
				return errors.Wrap(err, "deleting sandbox")
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Stack deleted.")

			return nil
		}),
	)
}
