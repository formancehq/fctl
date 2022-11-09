package cmd

import (
	"context"
	"fmt"

	"github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newSandboxDeleteCommand() *cobra.Command {
	const (
		sandboxNameFlag = "name"
	)

	return newMembershipCommand("delete [STACK_ID] | --name=[NAME]",
		withShortDescription("delete a sandbox"),
		withArgs(cobra.MaximumNArgs(1)),
		withStringFlag(sandboxNameFlag, "", "Sandbox to remove"),
		withPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		}),
		withRunE(func(cmd *cobra.Command, args []string) error {
			organization, err := fctl.FindOrganizationId(cmd.Context())
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := fctl.NewMembershipClientFromContext(cmd.Context())
			if err != nil {
				return err
			}

			var sandboxId string
			if len(args) == 1 {
				if viper.GetString(sandboxNameFlag) == "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				sandboxId = args[0]
			} else {
				if viper.GetString(sandboxNameFlag) == "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				sandboxs, _, err := apiClient.DefaultApi.ListStacks(context.Background(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing sandboxs")
				}
				for _, s := range sandboxs.Data {
					if s.Name == viper.GetString(sandboxNameFlag) {
						sandboxId = s.Id
						break
					}
				}
				if sandboxId == "" {
					return errors.New("sandbox not found")
				}
			}

			if _, err := apiClient.DefaultApi.DeleteStack(cmd.Context(), organization, sandboxId).Execute(); err != nil {
				return errors.Wrap(err, "deleting sandbox")
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Stack deleted.")

			return nil
		}),
	)
}
