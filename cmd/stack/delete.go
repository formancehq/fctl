package stack

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewDeleteCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)

	return cmdbuilder.NewMembershipCommand("delete [STACK_ID] | --name=[NAME]",
		cmdbuilder.WithShortDescription("Delete a sandbox"),
		cmdbuilder.WithArgs(cobra.MaximumNArgs(1)),
		cmdbuilder.WithStringFlag(stackNameFlag, "", "Sandbox to remove"),
		cmdbuilder.WithPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		}),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			organization, err := cmdbuilder.ResolveOrganizationID(cmd.Context(), cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := membership.NewClient(cmd.Context(), cfg)
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
